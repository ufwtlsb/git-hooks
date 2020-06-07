package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/urfave/cli"
)

func getGitRepoRoot() (string, error) {
	return gitExec("rev-parse --show-toplevel")
}

func getGitDirPath() (string, error) {
	return gitExec("rev-parse --git-dir")
}

func gitExec(args ...string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return gitExecWithDir(wd, args...)
}

func gitExecWithDir(dir string, args ...string) (string, error) {
	args = strings.Split(strings.Join(args, " "), " ")

	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	if out, err := cmd.Output(); err == nil {
		return string(bytes.Trim(out, "\n")), nil
	} else {
		return "", err
	}
}

func bind(f interface{}, args ...interface{}) func(c *cli.Context) {
	callable := reflect.ValueOf(f)
	arguments := make([]reflect.Value, len(args))
	for i, arg := range args {
		arguments[i] = reflect.ValueOf(arg)
	}
	return func(c *cli.Context) {
		callable.Call(arguments)
	}
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Download file from url.
// Downloaded file stored in temporary directory
func downloadFromUrl(url string) (fileName string, err error) {
	file, err := ioutil.TempFile(os.TempDir(), NAME)
	if err != nil {
		return
	}
	defer file.Close()

	fileName = file.Name()

	response, err := http.Get(url)
	if err != nil {
		return
	}
	defer response.Body.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return
	}

	return
}

func extract(fileName string) (tmpFileName string, err error) {
	file, err := ioutil.TempFile(os.TempDir(), NAME)
	if err != nil {
		return
	}
	defer file.Close()

	tmpFileName = file.Name()

	targz, err := os.Open(fileName)
	if err != nil {
		return
	}
	defer targz.Close()

	gr, err := gzip.NewReader(targz)
	if err != nil {
		return
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return tmpFileName, err
		}
		if hdr.Typeflag != tar.TypeDir {
			_, err = io.Copy(file, tr)
			if err != nil {
				return tmpFileName, err
			}
		}
	}
	return
}

func installBinary(src string) (err error) {
	dest, err := absExePath(os.Args[0])
	if err != nil {
		return
	}

	out, err := os.Create(dest)
	if err != nil {
		return
	}
	defer out.Close()

	err = out.Chmod(0755)
	if err != nil {
		return
	}

	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return
	}
	return
}

// return fullpath to executable file.
func absExePath(exe string) (name string, err error) {
	name = exe

	if name[0] == '.' {
		name, err = filepath.Abs(name)
		if err != nil {
			name = filepath.Clean(name)
		}
	} else {
		name, err = exec.LookPath(filepath.Clean(name))
	}
	if err != nil {
		return
	}
	// follow symlink
	fileinfo, err := os.Lstat(name)
	if err != nil {
		return
	}
	if fileinfo.Mode()&os.ModeSymlink != 0 {
		name, err = os.Readlink(name)
	}
	return
}

func isExecutable(info os.FileInfo) bool {
	if runtime.GOOS == "windows" {
		//windows 全部成功，因为取不到fullpath
		return true
		// 支持git for windows
		//参考git for windows的实现
		// if strings.HasSuffix(info.Name(), ".exe") {
		// 	return true
		// }
		// // a := info.(*os.fileStat)
		// // fmt.Println("fileStat is", a)
		// //windows下取到全path
		// // 用反射 os.fileStat
		// getType := reflect.TypeOf(info)
		// if getType.Kind() == reflect.Ptr {
		// 	getType = getType.Elem()
		// }
		// _, exist := getType.FieldByName("path")
		// if !exist {
		// 	//如果得不到path，就认为可以执行
		// 	return true
		// }
		// getValue := reflect.ValueOf(info).Elem()
		// //sf.Get()
		// //value := sf.Interface()
		// for i := 0; i < getType.NumField(); i++ {
		// 	field := getType.Field(i)
		// 	fmt.Printf("%s: %v =xxxv\n", field.Name, field.Type)
		// 	if field.Name == "path" {
		// 		value := getValue.Field(i).Interface()
		// 		fmt.Printf(" %v\n", value)
		// 	}

		// }

		// f, err := os.Open(info.Name())
		// if err != nil {
		// 	return false
		// }
		// sheBangHeader := make([]byte, 2)
		// n1, err := f.Read(sheBangHeader)
		// if err != nil {
		// 	return false
		// }
		// if n1 != 2 {
		// 	return false
		// }
		// sheBang := string(sheBangHeader[:n1])
		// fmt.Println("sheBang is", sheBang)
		// if sheBang == "#!" {
		// 	return true
		// }
		// return false
	}
	return info.Mode()&0111 != 0
}
