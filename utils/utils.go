package utils

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func CleanFilename(filename string, isFolder ...bool) string {
	regexExp := `[^a-zA-Z0-9#\._-]`
	if len(isFolder) > 0 && isFolder[0] {
		regexExp = `[^a-zA-Z0-9#\._/-]`
	}
	// Replace invalid characters with underscores
	re := regexp.MustCompile(regexExp)
	cleaned := re.ReplaceAllString(filename, "_")

	// Remove leading and trailing underscores
	cleaned = strings.Trim(cleaned, "_")

	// Remove leading dot (if any)
	cleaned = strings.TrimPrefix(cleaned, ".")

	// Make sure the result is not "." or ".."
	if cleaned == "" || cleaned == "." || cleaned == ".." {
		cleaned = "_"
	}

	return cleaned
}

func GetUserFolders(username string, id int) (string, string, string) {
	username = fmt.Sprintf("%s-%d", CleanFilename(username), id)
	homeFolder := filepath.Join("/hosting", username)
	mainFolder := filepath.Join(homeFolder, "LightningBot")

	return username, homeFolder, mainFolder
}

func UserExists(username string) bool {
	cmd := exec.Command("id", "-u", username)
	err := cmd.Run()
	return err == nil
}

func GetUIDGID(username string) (int, int, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return 0, 0, err
	}

	uid, err := strconv.ParseInt(u.Uid, 10, 32)
	if err != nil {
		return 0, 0, err
	}
	gid, err := strconv.ParseInt(u.Gid, 10, 32)
	if err != nil {
		return 0, 0, err
	}

	return int(uid), int(gid), nil
}

func CreateUser(username, home string) (int, int, error) {
	if !UserExists(username) {
		cmd := exec.Command("useradd", "-m", "-d", home, username)
		if err := cmd.Run(); err != nil {
			return 0, 0, err
		}
	}

	return GetUIDGID(username)
}

func CreateFolderIfNotExists(folder string, perm fs.FileMode, uid, gid int) error {
	if _, err := os.Stat(folder); !os.IsNotExist(err) {
		return nil
	}

	if err := os.MkdirAll(folder, perm); err != nil {
		return err
	}

	if err := os.Chown(folder, uid, gid); err != nil {
		return err
	}

	return nil
}

func IsDirEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
