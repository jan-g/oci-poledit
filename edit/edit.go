package edit

import (
	"bufio"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

// Edit writes a set of lines out to a tempfile then launches an editor to modify them.
// The resulting lines are returned to the caller.
// There is no assumption here that the file is not replaced in-situ by the editor.
func Edit(lines []string) ([]string, error) {
	tmpFile, err := ioutil.TempFile("", "edit-*")
	if err != nil {
		return lines, err
	}
	defer os.Remove(tmpFile.Name())
	writer := bufio.NewWriter(tmpFile)
	for _, s := range lines {
		_, err := writer.WriteString(s + "\n")
		if err != nil {
			tmpFile.Close()
			return lines, err
		}
	}
	if err := writer.Flush(); err != nil {
		return lines, err
	}
	tmpFile.Close()

	editor, ok := os.LookupEnv("VISUAL")
	if !ok {
		editor, ok = os.LookupEnv("EDITOR")
		if !ok {
			editor = "vi"
		}
	}

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return lines, err
	}

	tmpFile, err = os.Open(tmpFile.Name())
	if err != nil {
		return lines, err
	}
	defer tmpFile.Close()

	newLines := []string{}
	scanner := bufio.NewScanner(tmpFile)
	for scanner.Scan() {
		if line := strings.Trim(scanner.Text(), " "); line != "" {
			newLines = append(newLines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return lines, err
	}

	return newLines, nil
}
