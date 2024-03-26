package main

import (
	"bufio"
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

//go:embed luac.exe encrypt.exe
var embeddedFiles embed.FS

type actionFuncType func(string, []string) error

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	var file string
	fmt.Println("欢迎使用 JX3Toy_Encrypt 加密工具")
	fmt.Println("制作者: Cassian")
	fmt.Println("Github: https://github.com/Cassianvale")
	fmt.Println("JX3Toy: https://jx3toy.netlify.app/")
	fmt.Println("--------------------------------------------------")
	file = getFilePath(scanner)
	processFileWithSHA256Codes(file, encryptFile, scanner)

	for {
		continueCode := shouldContinue(scanner)
		if continueCode == 0 {
			break
		} else if continueCode == 1 {
			processFileWithSHA256Codes(file, encryptFile, scanner)
		} else if continueCode == 2 {
			file = getFilePath(scanner)
			processFileWithSHA256Codes(file, encryptFile, scanner)
		}
	}
}

func processFileWithSHA256Codes(src string, actionFunc actionFuncType, scanner *bufio.Scanner) {
	sha256Codes := getSHA256Codes(scanner)
	if err := processFile(src, sha256Codes, actionFunc); err != nil {
		fmt.Printf("加密失败: %v\n", err)
	} else {
		fmt.Printf("加密完成! \n")
	}
}

func getSHA256Codes(scanner *bufio.Scanner) []string {
	var sha256Codes []string
	fmt.Print("是否添加SHA256码? (y添加/回车跳过): ")
	if scanner.Scan() && scanner.Text() == "y" {
		for {
			fmt.Print("请输入SHA256码: ")
			if !scanner.Scan() {
				return sha256Codes
			}
			sha256Code := scanner.Text()

			// 如果用户输入为空或者为"y"，则跳出循环
			if sha256Code == "y" || strings.TrimSpace(sha256Code) == "" {
				break
			}
			sha256Codes = append(sha256Codes, sha256Code)
			fmt.Println("写入成功，请继续添加SHA256码，输入y或直接回车完成加密")
		}
	}
	return sha256Codes
}

func shouldContinue(scanner *bufio.Scanner) int {
	fmt.Print("输入数字键1继续加密当前文件，数字键2加密其他文件，按回车键退出终端: ")
	scanner.Scan()
	continueCode := scanner.Text()
	if continueCode == "1" {
		return 1
	} else if continueCode == "2" {
		return 2
	} else {
		return 0
	}
}

func getFilePath(scanner *bufio.Scanner) string {
	for {
		fmt.Print("请拖拽源文件: ")
		if !scanner.Scan() {
			return ""
		}
		file := scanner.Text()

		// 验证文件是否存在
		if _, err := os.Stat(file); os.IsNotExist(err) {
			fmt.Println("文件不存在，请重新输入")
			continue
		}

		// 验证文件是否为.lua文件
		if filepath.Ext(file) != ".lua" {
			fmt.Println("文件不是.lua文件，请重新输入")
			continue
		}

		return file
	}
}

func processFile(src string, sha256Codes []string, actionFunc actionFuncType) error {
	return actionFunc(src, sha256Codes)
}

func encryptFile(src string, sha256Codes []string) error {
	luacData, err := embeddedFiles.ReadFile("luac.exe")
	if err != nil {
		return err
	}
	luacPath := filepath.Join(os.TempDir(), "luac.exe")
	if err := os.WriteFile(luacPath, luacData, 0755); err != nil {
		return err
	}
	defer os.Remove(luacPath)

	// 读取源文件
	srcData, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	// 将源文件的内容转换为字符串
	srcStr := string(srcData)
	// 如果sha256Codes不为空，那么在 "function Main()" 后面插入新的代码
	if len(sha256Codes) > 0 {
		// 定位到 "function Main()" 的位置
		index := strings.Index(srcStr, "function Main()")
		if index == -1 {
			return fmt.Errorf("未找到 'function Main()' ")
		}

		// 在 "function Main()" 后面插入新的代码
		newCode := `
    if not tUsers[account()] then
        bigtext("非指定账号")
        return
    end
`
		// 将newCode转换为GBK格式
		gbkEncoder := simplifiedchinese.GBK.NewEncoder()
		newCodeGBK, _, err := transform.String(gbkEncoder, newCode)
		if err != nil {
			fmt.Println("转换为GBK格式时出错: ", err)
			return err
		}
		srcStr = srcStr[:index+len("function Main()")] + newCodeGBK + srcStr[index+len("function Main()"):]

		// 将修改后的内容转换回字节数组
		srcData = []byte(srcStr)

		// 创建一个新的 Lua 代码字符串，包含 tUsers 表和所有的 SHA256 码
		luaCode := "local tUsers = {\n"
		for _, code := range sha256Codes {
			luaCode += fmt.Sprintf("[\"%s\"] = true,\n", code)
		}
		luaCode += "}\n"

		// 将新的 Lua 代码字符串和源文件的内容合并
		srcData = append([]byte(luaCode), srcData...)
	}

	// 创建一个临时文件来存储合并后的内容
	tempFile, err := os.CreateTemp("", "temp")
	if err != nil {
		return err
	}

	if _, err := tempFile.Write(srcData); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	// 重命名临时文件为.lua文件
	luaFilePath := src[:len(src)-len(filepath.Ext(src))] + "_with_sha256.lua"

	// 创建新文件
	newFile, err := os.Create(luaFilePath)
	if err != nil {
		return err
	}
	defer newFile.Close()

	// 打开临时文件
	tempFile, err = os.Open(tempFile.Name())
	if err != nil {
		return err
	}
	defer tempFile.Close()

	// 复制临时文件内容到新文件
	if _, err := io.Copy(newFile, tempFile); err != nil {
		return err
	}
	// 关闭临时文件和新文件
	tempFile.Close()
	newFile.Close()

	// 删除临时文件
	if err := os.Remove(tempFile.Name()); err != nil {
		return err
	}

	// 使用 luac.exe 转换字节码
	luacOut := src[:len(src)-len(filepath.Ext(src))] + ".luac"
	cmd := exec.Command(luacPath, "-s", "-o", luacOut, luaFilePath)
	if err := cmd.Run(); err != nil {
		return err
	}

	// 使用 encrypt.exe 进行加密
	encryptData, err := embeddedFiles.ReadFile("encrypt.exe")
	if err != nil {
		return err
	}
	encryptPath := filepath.Join(os.TempDir(), "encrypt.exe")
	if err := os.WriteFile(encryptPath, encryptData, 0755); err != nil {
		return err
	}
	defer os.Remove(encryptPath)

	cmd = exec.Command(encryptPath, luacOut)
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command(encryptPath, luacOut)
	if err := cmd.Run(); err != nil {
		return err
	}

	// 删除转换后的 luac 字节码文件
	if err := os.Remove(luacOut); err != nil {
		return err
	}

	// 将扩展名替换为.luas
	luasOut := src[:len(src)-len(filepath.Ext(src))] + ".luas"

	// 打印出加密临时文件和加密后的文件名
	fmt.Println("加密临时文件: ", luaFilePath)
	fmt.Println("加密文件: ", luasOut)

	return nil
}
