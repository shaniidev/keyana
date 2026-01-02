package ui

import "fmt"

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
	Bold   = "\033[1m"
)

func Printf(color string, format string, a ...interface{}) {
	fmt.Printf(color+format+Reset, a...)
}

func Println(color string, a ...interface{}) {
	fmt.Print(color)
	fmt.Print(a...)
	fmt.Println(Reset)
}

func Info(format string, a ...interface{}) {
	fmt.Printf(Cyan+"[INFO] "+Reset+format+"\n", a...)
}

func Success(format string, a ...interface{}) {
	fmt.Printf(Green+"[+] "+Reset+format+"\n", a...)
}

func Error(format string, a ...interface{}) {
	fmt.Printf(Red+"[-] "+Reset+format+"\n", a...)
}

func Warning(format string, a ...interface{}) {
	fmt.Printf(Yellow+"[!] "+Reset+format+"\n", a...)
}

func Section(title string) {
	fmt.Printf("\n"+Bold+Blue+"=== %s ==="+Reset+"\n", title)
}
