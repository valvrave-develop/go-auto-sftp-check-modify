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
	NotModify   = iota //文件当前状态：未变更
	Modify             //文件当前状态：已修改
	Add                //文件当前状态：新增
	Delete             //文件当前状态：已删除(磁盘)
)

const (
	prefixSkipFile     = "skip_" //被忽略的文件
	defaultSliceLength = 50
)

type Directory struct {
	Dir   *Dir
	Index map[string]*Dir
}

func New() *Directory {
	return &Directory{
		Dir:   new(Dir),
		Index: make(map[string]*Dir),
	}
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
	fillDirIndex(d.Dir, d.Index)
	return nil
}

func (d *Directory) Open(path string) error {
	return traversalDir(path, d.Dir, d.Index)
}

func (d *Directory) CheckModify() ([]string, error) {
	return checkDirModify(d.Dir, d.Index)
}

func (d *Directory) Upload(client sftp.Sftp, localBaseDir, remoteBaseDir, localSep, remoteSep string, modify []string) error {
	for _, path := range modify {
		dir := d.Index[path]
		err := upload(client, localBaseDir, remoteBaseDir, localSep, remoteSep, dir, d.Index)
		if err != nil {
			return err
		}
	}
	return nil
}

type Dir struct {
	Name      string           `json:"dir_name"`   //目录文件名
	ChildDirs map[string]*Dir  `json:"dir_childs"` //目录包含的子目录
	Files     map[string]*File `json:"files"`      //目录包含的非目录文件
	Status    int              `json:"dir_status"` //目录当前的存在状态
}

type File struct {
	Name       string    `json:"file_name"`
	ModifyTime time.Time `json:"modify_time"`
	Status     int       `json:"file_status"`
}

func traversalDir(path string, dir *Dir, index map[string]*Dir) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("%s is not absolute path", path)
	}
	dirFile, err := os.OpenFile(path, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return fmt.Errorf("open file:%s failed, errMs:%v", path, err)
	}
	defer dirFile.Close()

	dirsContent, err := dirFile.Readdir(-1)
	if err != nil {
		return fmt.Errorf("traversal directory:%s failed, errMs:%v", path, err)
	}
	dir.Name = path
	dir.Status = Add
	dir.ChildDirs = make(map[string]*Dir)
	dir.Files = make(map[string]*File)

	for _, ele := range dirsContent {
		absolutePath := strings.Join([]string{path, ele.Name()}, string(filepath.Separator))
		if ele.IsDir() {
			childDir := new(Dir)
			if err := traversalDir(absolutePath, childDir, index); err != nil {
				return fmt.Errorf("traversal directory:%s failed, errMsg:%v", absolutePath, err)
			}
			dir.ChildDirs[ele.Name()] = childDir //存储子目录
		} else {
			file := &File{
				Name:       absolutePath,
				ModifyTime: ele.ModTime(),
				Status:     Add,
			}
			dir.Files[ele.Name()] = file //存储目录下非目录文件
		}
	}
	index[path] = dir //添加index，记录当前目录名对应的目录结构
	return nil
}

func fillDirIndex(dir *Dir, index map[string]*Dir) {
	for _, child := range dir.ChildDirs {
		fillDirIndex(child, index)
	}
	index[dir.Name] = dir
}

func checkDirModify(dir *Dir, index map[string]*Dir) ([]string, error) {
	if !filepath.IsAbs(dir.Name) {
		return nil, fmt.Errorf("%s is not absolute path", dir.Name)
	}
	dirFile, err := os.OpenFile(dir.Name, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return nil, fmt.Errorf("open file:%s failed, errMs:%v", dir.Name, err)
	}
	defer dirFile.Close()
	dirsContent, err := dirFile.Readdir(-1)
	if err != nil {
		return nil, fmt.Errorf("traversal directory:%s failed, errMs:%v", dir.Name, err)
	}
	modifyDir := make([]string, 0, defaultSliceLength)
	snapshoot := make(map[string]struct{})
	for key, _ :=  range dir.ChildDirs {
		snapshoot[key] = struct{}{}
	}
	for key, _ := range dir.Files {
		snapshoot[key] = struct{}{}
	}
	modirfyFlag := false
	for _, ele := range dirsContent {
		if ele.IsDir() {
			if _, ok := dir.ChildDirs[ele.Name()]; !ok { //不存在目录索引中，即新增目录，加载新的目录内容
				childDir := new(Dir)
				if err := traversalDir(strings.Join([]string{dir.Name, ele.Name()}, string(filepath.Separator)), childDir, index); err != nil {
					return nil, fmt.Errorf("traversal directory[%s] failed, errMsg:%v", ele.Name(), err)
				}
				dir.ChildDirs[ele.Name()] = childDir
				modirfyFlag = true
			} else {
				modify, err := checkDirModify(dir.ChildDirs[ele.Name()], index)
				if err != nil {
					return nil, fmt.Errorf("checkDirModify directory[%s] failed, errMsg:%v", ele.Name(), err)
				}
				modifyDir = append(modifyDir, modify...)
				delete(snapshoot, ele.Name())
			}
		} else {
			if file, ok := dir.Files[ele.Name()]; !ok {
				//新增文件
				file := &File{
					Name:       strings.Join([]string{dir.Name, ele.Name()}, string(filepath.Separator)),
					ModifyTime: ele.ModTime(),
					Status:     Add,
				}
				dir.Files[ele.Name()] = file
				modirfyFlag = true
			}else{
				if file.ModifyTime.Before(ele.ModTime()) {
					file.ModifyTime = ele.ModTime()
					file.Status = Modify
					modirfyFlag = true
				}
				delete(snapshoot, ele.Name())
			}
		}
	}
	for key, _ := range snapshoot  {
		if _, ok := dir.ChildDirs[key]; ok {
			dir.ChildDirs[key].Status = Delete
			modirfyFlag = true
		}
		if _, ok := dir.Files[key]; ok {
			dir.Files[key].Status = Delete
			modirfyFlag = true
		}
	}
	//判断staus==NotModify原因是若某个环节异常时，之前的改变不会回滚，导致Add的目录将不能正常上传
	//举例说明：
	//新增某个目录后，此时status=Add，之后在后续某个处理过程出错后直接返回，不会将其回滚
	//当下次轮询判断是否更改时，若status=Add的目录被更改时，若不加该判断时，status被更改为Modify时，
	//此时新增的目录将不能在远程创建（Add状态才会创建目录），从而导致上传失败
	if modirfyFlag && dir.Status == NotModify {
		dir.Status = Modify
	}
	if dir.Status != NotModify {
		modifyDir = append(modifyDir, dir.Name)
	}
	return modifyDir, nil
}

//Modify的目录由checkModify校验返回
//全量上传时，Modify的目录即Add的目录
//增量上传时，只处理Modify目录的子目录，除了Add变更，也只有Delete不再递归处理。
//增量上传时，要处理Modify目录下的所有更变文件，即Add，Modify，Delete
//删除的目录会变更上一级目录的状态，即改为Modify，继而交到Modify处理子目录中

func upload(client sftp.Sftp, localBaseDir, remoteBaseDir, localSep, remoteSep string, dir *Dir, index map[string]*Dir) error {
	localBaseDirLen := len(localBaseDir)
	remotePath := fmt.Sprintf("%s%s%s%s", remoteBaseDir, remoteSep, filepath.Base(localBaseDir), strings.Join(strings.Split(dir.Name[localBaseDirLen:], localSep), remoteSep))
	switch dir.Status {
	case Add:
		if err := client.Mkdir(remotePath); err != nil {
			return err
		}
		for _, child := range dir.ChildDirs {
			err := upload(client, localBaseDir, remoteBaseDir, localSep, remoteSep, child, index)
			if err != nil {
				return err
			}
		}
		for _, file := range dir.Files {
			if err := client.Put(file.Name, strings.Join([]string{remotePath, filepath.Base(file.Name)}, remoteSep)); err != nil {
				return err
			}
			file.Status = NotModify
		}
	case Modify:
		deleteKey := make([]string, 0, defaultSliceLength)
		for key, child := range dir.ChildDirs {
			switch child.Status {
			case Add:
				err := upload(client, localBaseDir, remoteBaseDir, localSep, remoteSep, child, index)
				if err != nil {
					return err
				}
			case Delete:
				if err := client.RemoveDirectory(strings.Join([]string{remotePath, filepath.Base(child.Name)}, remoteSep)); err != nil {
					return err
				}
				deleteKey = append(deleteKey, key)
			}
		}
		for _, key := range deleteKey {
			if _, ok := dir.ChildDirs[key]; ok {
				clearIndex(dir.ChildDirs[key], index)
				delete(dir.ChildDirs, key)
			}
		}
		deleteKey = deleteKey[:0]

		for key, file := range dir.Files {
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
				deleteKey = append(deleteKey, key)
			}
		}
		for _, key := range deleteKey {
			if _, ok := dir.ChildDirs[key]; ok {
				clearIndex(dir.ChildDirs[key], index)
				delete(dir.ChildDirs, key)
			}
			if _, ok := dir.Files[key]; ok {
				delete(dir.Files, key)
			}
		}
	}
	dir.Status = NotModify
	return nil
}
func clearIndex(dir *Dir, index map[string]*Dir) {
	delete(index, dir.Name)
	for _, child := range dir.ChildDirs {
		clearIndex(child, index)
	}
}
