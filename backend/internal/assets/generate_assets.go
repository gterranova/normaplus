//go:build ignore

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {
	// Base directories
	backendDir, _ := os.Getwd()
	// Since we run this from internal/assets via go generate, Getwd should be backend/internal/assets
	// But let's verify or use relative paths carefully.
	// go generate sets the working directory to the directory containing the file with the directive.

	frontendDir := filepath.Join("..", "..", "..", "frontend")
	distDir := filepath.Join(backendDir, "dist")

	fmt.Printf("Building frontend in: %s\n", frontendDir)

	// Determine npm command based on OS
	npmCmd := "npm"
	if runtime.GOOS == "windows" {
		npmCmd = "npm.cmd"
	}

	// 1. npm install
	if err := runCmd(npmCmd, []string{"install"}, frontendDir); err != nil {
		fmt.Printf("Error running npm install: %v\n", err)
		os.Exit(1)
	}

	// 2. npm run build
	if err := runCmd(npmCmd, []string{"run", "build"}, frontendDir); err != nil {
		fmt.Printf("Error running npm build: %v\n", err)
		os.Exit(1)
	}

	// 3. Clean dist directory
	fmt.Printf("Cleaning and preparing: %s\n", distDir)
	if err := os.RemoveAll(distDir); err != nil {
		fmt.Printf("Error removing dist directory: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(distDir, 0755); err != nil {
		fmt.Printf("Error creating dist directory: %v\n", err)
		os.Exit(1)
	}

	// 4. Copy out/* to dist/
	outDir := filepath.Join(frontendDir, "out")
	fmt.Printf("Copying assets from %s to %s\n", outDir, distDir)
	if err := copyDir(outDir, distDir); err != nil {
		fmt.Printf("Error copying assets: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Frontend assets generated and staged successfully!")
}

func runCmd(name string, args []string, dir string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		return copyFile(path, targetPath)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}

	return os.Chmod(dst, 0644)
}
