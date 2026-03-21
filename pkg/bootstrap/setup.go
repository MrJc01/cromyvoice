package bootstrap

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// EnsureEnvironment garante que o binário e modelos estejam instalados sem depender de usuário baixando zip no Browser.
func EnsureEnvironment() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(home, ".local", "share", "myvoice")
	os.MkdirAll(appDir, 0755)

	piperExe := filepath.Join(appDir, "piper", "piper")
	modelFile := filepath.Join(appDir, "pt_BR-faber-medium.onnx")
	jsonFile := filepath.Join(appDir, "pt_BR-faber-medium.onnx.json")

	downloadedSomething := false

	// Checa Piper Executável Estático
	if _, err := os.Stat(piperExe); os.IsNotExist(err) {
		fmt.Printf("\n⏳ [Bootstrap Máquina] Configurando Inteligência Artificial pela 1ª vez...\n")
		// Fetch Linux AMD64 Release via GitHub.
		if err := downloadAndExtractTarGz("https://github.com/rhasspy/piper/releases/download/2023.11.14-2/piper_linux_x86_64.tar.gz", appDir); err != nil {
			return appDir, fmt.Errorf("falha baixar binários IA nativos: %v", err)
		}
		downloadedSomething = true
	}

	// Checa Modelo Acústico Português do HuggingFace
	if _, err := os.Stat(modelFile); os.IsNotExist(err) {
		fmt.Printf("⏳ [Bootstrap Rede] Baixando Modelo Neural PT-BR (Faber-Medium ~60MB). Isso ocorrerá só 1 vez.\n")
		err := downloadFile("https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/pt/pt_BR/faber/medium/pt_BR-faber-medium.onnx?download=true", modelFile)
		if err != nil {
			return appDir, fmt.Errorf("falha ao injetar onnx: %v", err)
		}
		downloadedSomething = true
	}
	
	if _, err := os.Stat(jsonFile); os.IsNotExist(err) {
		err := downloadFile("https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/pt/pt_BR/faber/medium/pt_BR-faber-medium.onnx.json?download=true", jsonFile)
		if err != nil {
			return appDir, fmt.Errorf("falha injetar model spec json: %v", err)
		}
	}

	if downloadedSomething {
		fmt.Println("✅ [Soberania Digital] Ambiente isolado compilado no diretório:", appDir)
	}

	return appDir, nil
}

func downloadFile(url string, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status HTTP inexperado: %s", resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func downloadAndExtractTarGz(url string, destDir string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("falha no GET do Release TAR: %s", resp.Status)
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break 
		}
		if err != nil {
			return err
		}
		
		target := filepath.Join(destDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			// Preserva regras de execução (CHMOD) para o binário do Linux (piper)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		case tar.TypeSymlink:
			os.MkdirAll(filepath.Dir(target), 0755)
			os.Symlink(header.Linkname, target)
		}
	}
	return nil
}
