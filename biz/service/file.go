package service

import (
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	"github.com/cloudisk/biz/dal/query"
	"github.com/cloudisk/biz/model/common"
	"github.com/cloudisk/biz/model/gorm_gen"
	"gorm.io/gen"
	"gorm.io/gorm/clause"
)

func IsContainInt(items []int64, item int64) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}

func isContain(items []string, item string) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}

func getShareInfo(file *gorm_gen.File) (*gorm_gen.File, error) {
	if file.Share == 1 {
		return file, nil
	}
	pid := file.Pid
	for pid > 0 {
		result, err := query.Q.File.Where(query.File.Pid.Eq(int64(pid))).First()
		if err != nil {
			return nil, err
		}

		if result.Share == 1 {
			return result, nil
		}

		pid = result.Pid
	}
	return nil, nil
}

func getPermission(file *gorm_gen.File, userids []int64) int {
	if IsContainInt(userids, file.Userid) || IsContainInt(userids, file.CreatedID) {
		return 1000
	}
	row, err := getShareInfo(file)
	if err != nil {
		return -1
	}
	fileUser, err := query.Q.File_User.Where(query.File_User.FileID.Eq(row.ID)).Where(query.File_User.Userid.In(userids...)).Order(query.File_User.Permission).First()
	if err != nil {
		return -1
	}
	return int(fileUser.Permission)
}

func permissionFind(id int, user User, limit int) (*gorm_gen.File, error) {
	file, err := query.Q.File.Where(query.File.ID.Eq(int64(id))).First()
	if err != nil {
		return nil, errors.New("文件夹不存在或已被删除")
	}
	var userids []int64
	if isContain(user.Identity, "temp") {
		userids = []int64{int64(user.Userid)}
	} else {
		userids = []int64{0, int64(user.Userid)}
	}
	permission := getPermission(file, userids)
	if permission < limit {
		switch limit {
		case 1000:
			return nil, errors.New("仅限所有者或创建者操作")
		case 1:
			return nil, errors.New("没有修改写入权限")
		default:
			return nil, errors.New("没有查看访问权限")
		}
	}
	return file, err
}

func saveBeforePP(f *gorm_gen.File) bool {
	var pid int64 = f.Pid
	var pshare int64 = 0
	if f.Share > 0 {
		pshare = f.ID
	}

	var parentIds []int64
	for pid > 0 {
		parentIds = append(parentIds, pid)
		file, err := query.Q.File.Where(query.File.Pid.Eq(pid)).First()
		if err != nil {
			pid = 0
		}
		pid = file.Pid
		if file.Share > 0 {
			pshare = file.ID
		}
	}
	opids := f.Pids
	// Reverse the parent IDs to get the correct order
	for i, j := 0, len(parentIds)-1; i < j; i, j = i+1, j-1 {
		parentIds[i], parentIds[j] = parentIds[j], parentIds[i]
	}
	if len(parentIds) > 0 {
		// 反转数组
		for i, j := 0, len(parentIds)-1; i < j; i, j = i+1, j-1 {
			parentIds[i], parentIds[j] = parentIds[j], parentIds[i]
		}
		// 将数组转换为字符串
		f.Pids = "," + fmt.Sprintf("%v", parentIds) + ","
	} else {
		f.Pids = ""
	}
	f.Pshare = pshare
	// Save the file
	_, err := query.Q.File.Where(query.File.ID.Eq(f.ID)).Updates(f)
	if err != nil {
		return false
	}
	if f.Pids != opids {
		buf := make([]*gorm_gen.File, 0, 100)
		query.Q.File.Where(query.File.ID).FindInBatches(&buf, 100, func(tx gen.Dao, batch int) error {
			for _, result := range buf {
				saveBeforePP(result)
			}
			return nil
		})
	}
	return true
}

func HandleDuplicateName(f *gorm_gen.File) (newName string, err error) {
	// Check for existing file with the same name, pid, user ID, and extension
	var count int64
	_, err = query.Q.File.Where(query.File.Pid.Eq(f.Pid), query.File.Userid.Eq(f.Userid), query.File.Ext.Eq(f.Ext), query.File.Name.Eq(f.Name)).First()
	if err != nil {
		return "", err
	}
	if count == 0 {
		return "", err // No duplicate found
	}

	// Generate a new name
	var nextNum int = 2
	if matched, _ := regexp.MatchString(`(.*?)(\s+\(\d+\))*`, f.Name); matched {
		var preName string
		_, err := fmt.Sscanf(f.Name, "%s %d", &preName, &nextNum)
		if err != nil {
			return "", err
		}
		nextNum++

		// Check for existing file with the new name
		count, err = query.Q.File.Where(query.File.Pid.Eq(f.Pid), query.File.Userid.Eq(f.Userid), query.File.Ext.Eq(f.Ext), query.File.Name.Eq(f.Name), query.File.Name.Like(fmt.Sprintf("%s%", preName))).Count()
		if err != nil {
			return "", err
		}
		nextNum += int(count)
	}

	newName = fmt.Sprintf("%s (%d)", f.Name, nextNum)

	// Check for existing file with the newly generated name
	_, err = query.Q.File.Where(query.File.Pid.Eq(f.Pid), query.File.Userid.Eq(f.Userid), query.File.Ext.Eq(f.Ext), query.File.Name.Eq(f.Name)).First()
	if err != nil {
		nextNum = rand.Intn(9000) + 100
		newName = fmt.Sprintf("%s (%d)", f.Name, nextNum)
	}

	// f.Name = newName
	return newName, err
}

func Upload(user User, pid int, webkitRelativePath string, overwrite bool) (*common.File, error) {
	user_id := int64(user.Userid)
	if pid > 0 {
		count, _ := query.Q.File.Where(query.File.Pid.Eq(int64(pid))).Count()
		if count >= 300 {
			return nil, errors.New("每个文件夹里最多只能创建300个文件或文件夹")
		}
		row, err := permissionFind(pid, user, 1)
		if err != nil {
			return nil, err
		}
		user_id = row.Userid
	} else {
		count, _ := query.Q.File.Where(query.File.Userid.Eq(int64(user.Userid))).Where(query.File.Pid.Eq(0)).Count()
		if count >= 300 {
			return nil, errors.New("每个文件夹里最多只能创建300个文件或文件夹")
		}
	}

	dirs := strings.Split(webkitRelativePath, "/")

	for _, dirName := range dirs[1:] {
		if dirName == "" {
			continue
		}
		query.Q.Transaction(func(tx *query.Query) error {
			row, err := query.Q.File.Where(query.File.Pid.Eq(int64(pid)), query.File.Name.Eq(dirName)).Clauses(clause.Locking{Strength: "UPDATE"}).First()
			if err != nil {
				file := gorm_gen.File{Pid: int64(pid), Type: "folder", Name: dirName, Userid: user_id, CreatedID: int64(user.Userid)}
				file.Name, _ = HandleDuplicateName(&file)
				if saveBeforePP(&file) {
					query.Q.File.Create(&file)
				}
			}
			pid = int(row.Pid)
			return nil
		})
	}

	resp := &common.File{}

	return resp, nil
}
