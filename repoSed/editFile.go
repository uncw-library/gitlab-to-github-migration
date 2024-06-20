package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
)

func editFile(filepath string, needle string, replacement string) error {
	log.Printf("Starting\teditFile on: %s with %s and %s", filepath, needle, replacement)

	_, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		log.Printf("Skipping. File does not exist: %s", filepath)
		return nil
	}

	fileBytes, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}
	filetext := string(fileBytes)

	re := regexp.MustCompile(needle)
	matches := re.FindStringSubmatch(filetext)
	if len(matches) < 2 {
		// ok to not find the needle, continue to next file
		log.Printf("Info\tNot Found Needle '%s' in file '%s'", needle, filepath)
		return nil
	}

	log.Printf("Info\tMatches\t%v", matches)

	newtext := re.ReplaceAllString(filetext, fmt.Sprintf(replacement, matches[1]))

	info, err := os.Stat(filepath)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath, []byte(newtext), info.Mode())
	if err != nil {
		return err
	}
	return nil
}
