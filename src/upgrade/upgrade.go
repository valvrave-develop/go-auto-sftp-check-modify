package main

import (
	"dir"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

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


func main(){
	fp, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}
	defer fp.Close()
	decode := json.NewDecoder(fp)
	oldDir := new(DirectoryStruct)
	if err := decode.Decode(oldDir); err != nil {
		log.Fatalln(err)
	}

	newDir := new(dir.Dir)
	upgrade(oldDir, newDir)

	newFp, err := os.Create(fmt.Sprint(os.Args[1], ".temp"))
	if err != nil {
		log.Fatalln(err)
	}
	defer newFp.Close()
	encode := json.NewEncoder(newFp)
	if err := encode.Encode(newDir); err != nil {
		log.Fatalln(err)
	}
}

func upgrade(old *DirectoryStruct, newDir *dir.Dir) {
	newDir.Name = old.DirName
	newDir.Status = old.Status
	newDir.ChildDirs = make(map[string]*dir.Dir)
	newDir.Files = make(map[string]*dir.File)
	for _, oldChild := range old.DirChild {
		newChild := new(dir.Dir)
		upgrade(oldChild, newChild)
		newDir.ChildDirs[filepath.Base(oldChild.DirName)] = newChild
	}

	for _, file := range old.File {
		newDir.Files[filepath.Base(file.Name)] = &dir.File{
			Name:       file.Name,
			ModifyTime: file.ModifyTime,
			Status:     file.Status,
		}
	}
}