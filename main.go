package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	option := os.Args[1]

	switch option {
	case "create", "c":
		if len(os.Args) < 3 {
			fmt.Println("Usage: " + os.Args[0] + " create <name>")
			os.Exit(1)
		}

		projectName := os.Args[2]
		handleCreate(projectName)
	case "run", "r":
		if len(os.Args) < 3 {
			fmt.Println("Executando em modo UnspecifiedFile (podem ocorrer erros!)")
		}

		val, exists := getItem(os.Args, 2)
		handleRun(val, exists)
	case "add":
		if len(os.Args) < 3 {
			fmt.Println("Usage: " + os.Args[0] + " add <name>")
			os.Exit(1)
		}

		pkgs := os.Args[2:]
		handleAdd(pkgs)
	case "remove", "rm":
		if len(os.Args) < 3 {
			fmt.Println("Usage: " + os.Args[0] + " remove <name>")
			os.Exit(1)
		}

		name := os.Args[2:]
		handleRemove(name)
	case "init":
		handleInit()
	case "install":
		handleInstall()
	case "clean":
		handleClean()
	default:
		fmt.Println("Comando '" + option + "' não encontrado")
		printHelp()
		os.Exit(1)
	}
}

func handleCreate(project string) {
	err := os.MkdirAll(project, 0755)
	if err != nil {
		fmt.Println("Erro ao criar projeto Python:", err)
		return
	}

	cmd := exec.Command("python3", "-m", "venv", ".venv")
	cmd.Dir = project

	isApi := in("--api", os.Args)

	_, err = cmd.Output()
	if err != nil {
		fmt.Println("Aviso: Não consegui criar venv")
		fmt.Println("Você pode criar manualmente depois: python -m venv .venv")

		if isApi {
			fmt.Println("Aviso: Não e possível criar uma API sem um venv")
			fmt.Println("O projeto sera criado como um projeto normal!")
			isApi = false
		}
	}

	fmt.Println("Venv criado com sucesso!")

	content := []byte(`def main() -> None:
	print('Hello, World!')

if __name__ == '__main__':
	main()`)

	if isApi {
		content, err = apiHelper(project, cmd, err, content)
		if err != nil {
			fmt.Println("Erro ao chamar apiHelper:", err)
		}
	}

	mainFilePath := filepath.Join(project, "main.py")

	err = os.WriteFile(mainFilePath, content, 0644)
	if err != nil {
		fmt.Println("Erro ao criar:", err)
		return
	}

	fmt.Println("Arquivo main criado com sucesso!")
	fmt.Println("Projeto Python " + project + " criado com sucesso!")
}

func apiHelper(project string, cmd *exec.Cmd, err error, content []byte) ([]byte, error) {
	venvPython := filepath.Join(".venv", "bin", "python")
	cmd = exec.Command(venvPython, "-m", "pip", "install", "fastapi", "uvicorn")
	cmd.Dir = project

	_, err = cmd.Output()
	if err != nil {
		fmt.Println("Erro ao criar venv:", err)
	}

	fmt.Println("FastAPI baixado com sucesso!")

	content = []byte(`from fastapi import FastAPI

app = FastAPI()

@app.get('/')
def index():
	return {'Hello': 'World!'}`)

	runFile := filepath.Join(project, "run.sh")
	runFileContent := []byte(`#!/bin/bash
if [ -d ".venv" ]; then
    source .venv/bin/activate
fi

python -m uvicorn main:app --reload`)

	err = os.WriteFile(runFile, runFileContent, 0755)
	if err != nil {
		fmt.Println("Erro ao criar run.sh:", err)
		return content, err
	}

	fmt.Println("run.sh criado com sucesso! Rode com ./run.sh")

	requirementsFile := filepath.Join(project, "requirements.txt")
	requirementsContent := []byte(`fastapi
uvicorn
`)
	err = os.WriteFile(requirementsFile, requirementsContent, 0644)
	if err != nil {
		fmt.Println("Erro ao criar requirements.txt:", err)
		return content, err
	}

	return content, err
}

func handleRun(file string, specificFile bool) {
	if specificFile {
		cmd := exec.Command("python3", file)

		out, err := cmd.Output()
		if err != nil {
			fmt.Println("Um erro ocorreu:", err)
			fmt.Println(string(out))
			return
		}

		fmt.Print(string(out))
	} else {
		entries, err := os.ReadDir(".")
		if err != nil {
			fmt.Println("Erro:", err)
			return
		}

		for _, e := range entries {
			if filepath.Ext(e.Name()) == ".py" {
				cmd := exec.Command("python3", e.Name())

				out, err := cmd.Output()
				if err != nil {
					fmt.Println("Erro ao rodar "+e.Name()+":", err)
				}
				fmt.Print(string(out))
			}
		}
	}
}

func handleAdd(pkgs []string) {
	_, err := os.Stat(".venv")
	hasVenv := err == nil
	_, err = os.Stat("requirements.txt")
	hasRequirements := err == nil

	if hasVenv {
		venvPython := filepath.Join(".venv", "bin", "python")

		if hasRequirements {
			data, err := os.ReadFile("requirements.txt")
			if err != nil {
				fmt.Println("Erro ao ler requirements.txt:", err)
				return
			}

			for _, pkg := range pkgs {
				if strings.Contains(string(data), pkg) {
					fmt.Println("O pacote " + pkg + " já está no requirements.txt")
				} else {
					cmd := exec.Command(venvPython, "-m", "pip", "install", pkg)

					_, err = cmd.Output()
					if err != nil {
						fmt.Println("Erro ao instalar "+pkg+":", err)
						return
					}

					fmt.Println("Pacote " + pkg + " adicionado com sucesso!")

					f, err := os.OpenFile("requirements.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if err != nil {
						fmt.Println("Erro ao abrir requirements.txt:", err)
						return
					}

					_, err = f.WriteString(pkg + "\n")
					if err != nil {
						fmt.Println("Erro ao escrever requirements.txt:", err)
						return
					}

					err = f.Close()
					if err != nil {
						fmt.Println("Erro ao fechar requirements.txt:", err)
						return
					}

					fmt.Println("Pacote " + pkg + " adicionado ao requirements.txt com sucesso!")
				}
			}
		} else {
			for _, pkg := range pkgs {
				cmd := exec.Command(venvPython, "-m", "pip", "install", pkg)

				_, err = cmd.Output()
				if err != nil {
					fmt.Println("Erro ao instalar "+pkg+":", err)
					return
				}

				fmt.Println("Pacote " + pkg + " baixado com sucesso!")
			}
		}
	} else {
		for _, pkg := range pkgs {
			cmd := exec.Command("python", "-m", "pip", "install", pkg)
			_, err := cmd.Output()
			if err != nil {
				fmt.Println("Erro ao baixar "+pkg+" :", err)
			} else {
				fmt.Println("Pacote " + pkg + " baixado com sucesso!")
			}
		}
	}
}

func handleRemove(names []string) {
	_, err := os.Stat(".venv")
	hasVenv := err == nil
	_, err = os.Stat("requirements.txt")
	hasRequirements := err == nil

	if hasVenv {
		venvPython := filepath.Join(".venv", "bin", "python")

		for _, pkg := range names {
			cmd := exec.Command(venvPython, "-m", "pip", "uninstall", "-y", pkg)
			_, err := cmd.Output()
			if err != nil {
				fmt.Println("Erro ao remover "+pkg+":", err)
			} else {
				fmt.Println("Pacote " + pkg + " removido com sucesso!")
			}
		}

		if hasRequirements {
			file, err := os.Open("requirements.txt")
			if err != nil {
				fmt.Println("Erro ao ler requirements.txt:", err)
			}

			var lines []string
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				shouldKeep := true

				for _, pkg := range names {
					if strings.TrimSpace(line) == pkg {
						shouldKeep = false
						break
					}
				}

				if shouldKeep && line != "" {
					lines = append(lines, line)
				}
			}

			content := strings.Join(lines, "\n")
			content = content + "\n"

			err = os.WriteFile("requirements.txt", []byte(content), 0644)
			if err != nil {
				fmt.Println("Erro ao escrever requirements.txt:", err)
				return
			}
		}
	} else {
		for _, pkg := range names {
			cmd := exec.Command("python", "-m", "pip", "remove", pkg)
			_, err := cmd.Output()
			if err != nil {
				fmt.Println("Erro ao remover "+pkg+":", err)
			} else {
				fmt.Println("Pacote " + pkg + " removido com sucesso!")
			}
		}
	}
}

func handleInit() {
	var imports []string

	hasVenv := hasFile(".venv")
	var venvPython string
	var pkgs []string

	if !(hasFile("requirements.txt")) {
		f, err := os.Create("requirements.txt")
		if err != nil {
			fmt.Println("Erro ao criar requirements.txt:", err)
		}

		err = f.Close()
		if err != nil {
			fmt.Println("Erro ao fechar requirements.txt:", err)
		}
	}

	if !hasVenv {
		cmd := exec.Command("python", "-m", "venv", ".venv")
		_, err := cmd.Output()
		if err != nil {
			fmt.Println("Erro ao criar venv:", err)
		} else {
			fmt.Println("Venv criado com sucesso!")
			hasVenv = true
			venvPython = filepath.Join(".venv", "bin", "python")
		}
	}

	if hasFile("main.py") {
		file, err := os.Open("main.py")
		if err != nil {
			fmt.Println("Erro ao ler main.py:", err)
		}

		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				fmt.Println("Erro ao fechar main.py:", err)
			}
		}(file)

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			if strings.HasPrefix(line, "import ") {
				imports = append(imports, line[len("import "):])
			}

			if strings.HasPrefix(line, "from ") {
				parts := strings.Split(line[len("from "):], " ")
				imports = append(imports, parts[0])
			}
		}

		if err := scanner.Err(); err != nil {
			fmt.Println("Erro ao ler main.py:", err)
			return
		}

	}

	for pkg := range imports {
		pkgs = append(pkgs, imports[pkg])
	}

	err := os.WriteFile("requirements.txt", []byte(strings.Join(pkgs, "\n")), 0644)
	if err != nil {
		fmt.Println("Erro ao escrever os requerimentos no requirements.txt:", err)
	} else {
		fmt.Println("Pacotes adicionados ao requirements.txt com sucesso!")
	}

	cmd := exec.Command(venvPython, "-m", "pip", "install", "-r", "requirements.txt")
	_, err = cmd.Output()
	if err != nil {
		fmt.Println("Erro ao baixar requerimentos:", err)
	}
}

func handleInstall() {
	hasVenv := hasFile(".venv")

	if !(hasFile("requirements.txt")) {
		fmt.Println("Não e possível usar o comando install sem um requirements.txt")
		fmt.Println("Use o comando malo init para ter um requirements.txt")
		return
	}

	if hasVenv {
		venvPython := filepath.Join(".venv", "bin", "python")

		cmd := exec.Command(venvPython, "-m", "pip", "install", "-r", "requirements.txt")
		out, err := cmd.Output()
		if err != nil {
			fmt.Println("Erro ao instalar pacotes do requirements.txt:", err)
			fmt.Println(string(out))
		}
	} else {
		cmd := exec.Command("python", "-m", "pip", "install", "-r", "requirements.txt")
		out, err := cmd.Output()
		if err != nil {
			fmt.Println("Erro ao instalar pacotes do requirements.txt:", err)
		}
		fmt.Println(string(out))
	}
}

func handleClean() {
	if hasFile("__pycache__") {
		err := os.RemoveAll("__pycache__")
		if err != nil {
			fmt.Println("Erro ao remover __pycache__:", err)
		}
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		fmt.Println("Erro ao ler o diretório atual:", err)
		return
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".pyc" {
			err = os.Remove(entry.Name())
			if err != nil {
				fmt.Println("Erro ao remover "+entry.Name()+":", err)
			} else {
				fmt.Println("Pacote " + entry.Name() + " removido com sucesso!")
			}
		}
	}
}

func getItem[T any](s []T, i int) (T, bool) {
	if i >= 0 && i < len(s) {
		return s[i], true
	}
	var zero T
	return zero, false
}

func in(s string, t []string) bool {
	for _, i := range t {
		if i == s {
			return true
		}
	}

	return false
}

func printHelp() {
	fmt.Println(`Como usar Mal0 Helper:
malo <comando> <args>

Comandos:
	create/c  -> Cria um novo projeto
	run/r 	  -> Roda um arquivo ou projeto
	add		  -> Adiciona pacotes ao projeto
	remove/rm -> Remove pacotes do projeto

Exemplos:
	malo create <linguagem> <nome do projeto>
	malo run <arquivo ou em branco para modo UnspecifiedFile>
	malo add <nomes dos pacotes>
	malo remove <nomes dos pacotes>`)
}

func hasFile(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}
