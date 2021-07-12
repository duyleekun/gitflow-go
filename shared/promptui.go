package shared

import (
	"github.com/manifoldco/promptui"
	"github.com/manifoldco/promptui/list"
)

func PromptString(promptMessage string) string {
	//validate := func(input string) error {
	//	_, err := strconv.ParseFloat(input, 64)
	//	if err != nil {
	//		return errors.New("Invalid number")
	//	}
	//	return nil
	//}
	//
	prompt := promptui.Prompt{
		Label: promptMessage,
		//Validate: validate,
	}

	result, err := prompt.Run()

	if err != nil {
		PrintVerbose("Prompt failed %v\n", err)
		return ""
	}

	return result
}

func PromptSelect(label string, length int, searcher list.Searcher, nameMapper func(int) string) int {
	var projectNames []string

	for i := 0; i < length; i++ {
		projectNames = append(projectNames, nameMapper(i))
	}

	prompt := promptui.Select{
		Label:             label,
		Items:             projectNames,
		Searcher:          searcher,
		StartInSearchMode: true,
	}

	selectedIndex, _, err := prompt.Run()

	if err != nil {
		PrintVerbose("Prompt failed %v\n", err)
		return -1
	}

	return selectedIndex
}
