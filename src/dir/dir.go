package dir

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sftp"
	"strings"
	"time"
)

//文件状态
const (
	NotModify = iota //文件当前状态：未变更
	Modify           //文件当前状态：已修改
	Add              //文件当前状态：新增
	Delete           //文件当前状态：已删除(磁盘)
	ShiftDelete      //彻底删除文件（缓存记录）
)

const (
	prefixSkipFile = "skip_"   //被忽略的文件
	defaultSliceLength = 50
)


type Directory struct {
	DirMap     map[string]*DirectoryStruct
	Dir        *DirectoryStruct
}

func New() *Directory{
	return &Directory{
		DirMap:   make(map[string]*DirectoryStruct),
		Dir:      new(DirectoryStruct),
	}
}

type DirectoryStruct struct {
	DirName string                     `json:"dir_name"`            //目录文件名
	ModifyTime time.Time               `json:"dir_modify_time"`     //目录的修改时间
	DirChild []*DirectoryStruct        `json:"dir_child"`           //目录包含的子目录
	File []*FileStruct                 `json:"file"`                //目录包含的非目录文件
	Status int                         `json:"dir_status"`          //目录当前的存在状态
	ExistFile map[string]bool          `json:"-"`                   //文件存在状态，与ExistFlag一起校验对应文件是否存在
	ExistFlag bool                     `json:"-"`                   //文件存在状态值，true和false
}

type FileStruct struct {
	Name string `json:"name"`
	ModifyTime time.Time `json:"modify_time"`
	Status int  `json:"file_status"`
}

func (d *Directory) EncodeJson(w io.Writer) error {
	encode := json.NewEncoder(w)
	if err := encode.Encode(d.Dir); err != nil {
		return err
	}
	return nil
}

func (d *Directory) DecodeJson(r io.Reader) error {
	decode := json.NewDecoder(r)
	if err := decode.Decode(d.Dir); err != nil {
		return err
	}
	fillDirIndex(d.Dir, d.DirMap)
	return nil
}

func (d *Directory) Open(dir string) error {
	return traversalDir(dir, d.Dir, d.DirMap)
}

func (d *Directory) CheckModify() ([]string, error){
	return checkDirModify(d.Dir, d.DirMap)
}

func (d *Directory) Upload(client sftp.Sftp, localBaseDir,remoteBaseDir,localSep, remoteSep string, modify []string) error {
	for _, path := range modify {
		dir := d.DirMap[path]
		if dir.Status == NotModify {
			continue
		}
		err := upload(client,localBaseDir,remoteBaseDir,localSep,remoteSep,dir)
		clear(dir, d.DirMap) //不管upload是否失败，都要clear一次
		if err != nil {
			return err
		}
	}
	return nil
}

func traversalDir(srcDir string, dir *DirectoryStruct, dirIndex map[string]*DirectoryStruct) error{
	if !filepath.IsAbs(srcDir) {
		return fmt.Errorf("%s is not absolute path", srcDir)
	}
	dirFile, err := os.OpenFile(srcDir, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return fmt.Errorf("open file:%s failed, errMs:%v", srcDir, err)
	}
	defer dirFile.Close()

	dirsContent, err := dirFile.Readdir(-1)
	if err != nil {
		return fmt.Errorf("directory traversal:%s failed, errMs:%v", srcDir, err)
	}
	dir.DirName = srcDir
	dir.Status = Add
	if dir.DirChild == nil {
		dir.DirChild = make([]*DirectoryStruct, 0, defaultSliceLength)
	}
	if dir.File == nil {
		dir.File = make([]*FileStruct, 0, defaultSliceLength)
	}
	if dir.ExistFile == nil {
		dir.ExistFile = make(map[string]bool)
	}
	dir.ExistFlag = true
	for _, ele := range dirsContent {
		if strings.Index(ele.Name(),prefixSkipFile) == 0 {
			continue
		}
		absolutePath := fmt.Sprintf("%s%s%s",srcDir,string(filepath.Separator),ele.Name())
		if ele.IsDir(){
			childDir := new(DirectoryStruct)
			childDir.ModifyTime = ele.ModTime()
			if err := traversalDir(absolutePath, childDir, dirIndex); err != nil {
				return fmt.Errorf("traversal directory failed, errMsg:%v", err)
			}
			dir.DirChild = append(dir.DirChild, childDir)        //存储子目录
		}else{
			file := &FileStruct{
				Name:absolutePath,
				ModifyTime:ele.ModTime(),
				Status:Add,
			}
			dir.File = append(dir.File, file)                    //存储目录下非目录文件
		}
		dir.ExistFile[absolutePath] = dir.ExistFlag              //标记当前目录下所有文件
	}
	dirIndex[srcDir] = dir                                       //添加index，记录当前目录名对应的目录结构
	return nil
}

func fillDirIndex(dir *DirectoryStruct, dirIndex map[string]*DirectoryStruct) {
	if dir.ExistFile == nil {
		dir.ExistFile = make(map[string]bool)
	}
	dir.ExistFlag = true
	for _, child := range dir.DirChild {
		dir.ExistFile[child.DirName] = dir.ExistFlag
		fillDirIndex(child, dirIndex)
	}
	for _, file := range dir.File {
		dir.ExistFile[file.Name] = dir.ExistFlag
	}
	dirIndex[dir.DirName] = dir
}

func  checkDirModify(dir *DirectoryStruct, dirIndex map[string]*DirectoryStruct) ([]string, error){
	if !filepath.IsAbs(dir.DirName) {
		return nil, fmt.Errorf("%s is not absolute path", dir.DirName)
	}
	dirFile, err := os.OpenFile(dir.DirName, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return nil, fmt.Errorf("open file:%s failed, errMs:%v", dir.DirName, err)
	}
	defer dirFile.Close()
	dirsContent, err := dirFile.Readdir(-1)
	if err != nil {
		return nil, fmt.Errorf("directory traversal:%s failed, errMs:%v", dir.DirName, err)
	}
	modifyDir := make([]string, 0, defaultSliceLength)
	dir.ExistFlag = !dir.ExistFlag  //表示新的一轮check
	var e error = nil
	defer func(){
		if e != nil {
			dir.ExistFlag = !dir.ExistFlag
			for key, _ := range dir.ExistFile {
				dir.ExistFile[key] = dir.ExistFlag
			}
		}
	}()
	for _, ele := range dirsContent {
		if strings.Index(ele.Name(),prefixSkipFile) == 0 {
			continue
		}
		absolutePath := fmt.Sprintf("%s%s%s",dir.DirName,string(filepath.Separator),ele.Name())
		dir.ExistFile[absolutePath] = dir.ExistFlag
		if ele.IsDir(){
			if _, ok := dirIndex[absolutePath]; !ok {  //不存在目录索引中，即新增目录，加载新的目录内容
				childDir := new(DirectoryStruct)
				childDir.ModifyTime = ele.ModTime()
				if err := traversalDir(absolutePath, childDir, dirIndex); err != nil {
					e = err
					return nil, fmt.Errorf("traversal directory[%s] failed, errMsg:%v", absolutePath, err)
				}
				dir.DirChild = append(dir.DirChild, childDir)
				if dir.Status == NotModify {
					dir.Status = Modify
				}
			}else{
				dirIndex[absolutePath].ModifyTime = ele.ModTime()
				modify, err := checkDirModify(dirIndex[absolutePath], dirIndex)
				if err != nil {
					e = err
					return nil, fmt.Errorf("checkDirModify directory[%s] failed, errMsg:%v", absolutePath, err)
				}
				modifyDir = append(modifyDir, modify...)
			}
		}else{
			continueFlag := false
			for _, file := range dir.File {
				if absolutePath == file.Name { //检查已存在的文件是否发生变化
					if file.ModifyTime.Before(ele.ModTime()) {
						file.ModifyTime = ele.ModTime()
						file.Status = Modify
						if dir.Status == NotModify {
							dir.Status = Modify
						}
					}
					continueFlag = true
					break
				}
			}
			if continueFlag {
				continue
			}
			//新增文件
			file := &FileStruct{
				Name:absolutePath,
				ModifyTime:ele.ModTime(),
				Status:Add,
			}
			dir.File = append(dir.File, file)
			if dir.Status == NotModify {
				dir.Status = Modify
			}
		}
	}
	deleteKey := make([]string, 0, defaultSliceLength)
	for file, value := range dir.ExistFile {
		if value == dir.ExistFlag { //与当前ExistFlag不一致时，表示此时遍历未遍历到该文件，即不一致对应的文件已删除
			continue
		}
		deleteKey = append(deleteKey, file) //记录当前遍历已经不存在的文件
		if _, ok := dirIndex[file]; ok {
			//删除的目录
			//注意：上述处理中只会递归检测仍然存在的目录，对于已删除的目录不会检测，因此不会出现递归目录中存在多个删除目录事件
			//即被删除目录可以直接删除，不用检测其上级目录是否存在
			dirIndex[file].Status = Delete
			if dir.Status == NotModify {
				dir.Status = Modify
			}
		}else{
			for _, f := range dir.File {
				if f.Name == file{
					f.Status = Delete
					if dir.Status == NotModify {
						dir.Status = Modify
					}
				}
			}
		}
	}
	for _, file := range deleteKey {
		delete(dir.ExistFile, file)
	}
	if dir.Status != NotModify {
		modifyDir = append(modifyDir, dir.DirName)
	}
	return modifyDir, nil
}
//Modify的目录由checkModify校验返回
//全量上传时，Modify的目录即Add的目录
//增量上传时，只处理Modify目录的子目录，除了Add变更，也只有Delete不再递归处理。
//增量上传时，要处理Modify目录下的所有更变文件，即Add，Modify，Delete
//删除的目录会变更上一级目录的状态，即改为Modify，继而交到Modify处理子目录中

func upload(client sftp.Sftp, localBaseDir,remoteBaseDir,localSep, remoteSep string, dir *DirectoryStruct) error {
	localBaseDirLen := len(localBaseDir)
	remotePath := fmt.Sprintf("%s%s%s%s", remoteBaseDir, remoteSep, filepath.Base(localBaseDir), strings.Join(strings.Split(dir.DirName[localBaseDirLen:], localSep), remoteSep))
	switch dir.Status {
	case Add:
		if err := client.Mkdir(remotePath); err != nil {
			return err
		}
		for _, nextDir := range dir.DirChild {
			err := upload(client, localBaseDir, remoteBaseDir, localSep, remoteSep, nextDir)
			if err != nil {
				return err
			}
		}
		for _, file := range dir.File {
			if err := client.Put(file.Name, strings.Join([]string{remotePath, filepath.Base(file.Name)}, remoteSep)); err != nil {
				return err
			}
			file.Status = NotModify
		}
		dir.Status = NotModify
	case Modify:
		for _, nextDir := range dir.DirChild {
			switch nextDir.Status {
			case Add:
				err := upload(client, localBaseDir, remoteBaseDir, localSep, remoteSep, nextDir)
				if err != nil {
					return err
				}
			case Delete:
				if err := client.RemoveDirectory(strings.Join([]string{remotePath, filepath.Base(nextDir.DirName)}, remoteSep)); err != nil {
					return err
				}
				nextDir.Status = ShiftDelete
			}
		}
		for _, file := range dir.File {
			switch file.Status {
			case Add:
				fallthrough
			case Modify:
				if err := client.Put(file.Name, strings.Join([]string{remotePath, filepath.Base(file.Name)}, remoteSep)); err != nil {
					return err
				}
				file.Status = NotModify
			case Delete:
				if err := client.Remove(strings.Join([]string{remotePath, filepath.Base(file.Name)}, remoteSep)); err != nil {
					return err
				}
				file.Status = ShiftDelete
			}
		}
		dir.Status = NotModify
	}
	return nil
}

func clear(dir *DirectoryStruct, dirIndex map[string]*DirectoryStruct) {
	for _, nextDir := range dir.DirChild {
		clear(nextDir, dirIndex)
	}
	total := len(dir.File)
	i := 0
	for {
		if i >= total {
			break
		}
		if dir.File[i].Status == ShiftDelete {
			total -= 1
			dir.File[i] = dir.File[total]
			dir.File = dir.File[:total]
		}else{
			i++
		}
	}
	total = len(dir.DirChild)
	i = 0
	for {
		if i >= total {
			break
		}
		if dir.DirChild[i].Status == ShiftDelete {
			total -= 1
			//清理目录索引
			deleteIndexKey := make([]string, 0, defaultSliceLength)
			for key, _ := range dirIndex {
				if strings.Index(key, dir.DirChild[i].DirName) == 0 {
					deleteIndexKey = append(deleteIndexKey, key)
				}
			}
			for _, key := range deleteIndexKey {
				delete(dirIndex, key)
			}
			dir.DirChild[i] = dir.DirChild[total]
			dir.DirChild = dir.DirChild[:total]
		}else{
			i++
		}
	}
}