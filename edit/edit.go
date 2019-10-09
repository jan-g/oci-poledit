package edit

import (
	"bufio"
	"encoding/json"
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

	if err := LaunchEditor(tmpFile.Name()); err != nil {
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

// LaunchEditor will open a file in the user's chosen editor.
func LaunchEditor(filename string) error {
	editor, ok := os.LookupEnv("VISUAL")
	if !ok {
		editor, ok = os.LookupEnv("EDITOR")
		if !ok {
			editor = "vi"
		}
	}

	cmd := exec.Command(editor, filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Json writes a json-encodable object out to a tempfile then launches an editor to modify it.
// The resulting json is unmarshalled back into the same struct and returned to the caller.
// There is no assumption here that the file is not replaced in-situ by the editor.
// Usage: Json(&myStruct)
// This must be called with a pointer to the target structure in order to preserve type information;
// otherwise you'll get a map[string]interface{} back.
func Json(item interface{}) (interface{}, error) {
	tmpFile, err := ioutil.TempFile("", "edit-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())

	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(item); err != nil {
		tmpFile.Close()
		return nil, err
	}

	tmpFile.Close()

	if err := LaunchEditor(tmpFile.Name()); err != nil {
		return nil, err
	}

	tmpFile, err = os.Open(tmpFile.Name())
	if err != nil {
		return nil, err
	}
	defer tmpFile.Close()

	decoder := json.NewDecoder(tmpFile)
	if err := decoder.Decode(&item); err != nil {
		return nil, err
	}

	return item, nil
}
