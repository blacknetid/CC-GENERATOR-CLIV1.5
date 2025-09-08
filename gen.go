package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CCGenerator struct {
	apikey     string
	baseURL    string
	resultsDir string
	client     *http.Client
	mutex      sync.Mutex
}

type CCData struct {
	Date   string `json:"date"`
	CC     string `json:"cc"`
	Month  string `json:"month"`
	Year   string `json:"year"`
	CVV    string `json:"cvv"`
	Scheme string `json:"scheme"`
}

type APIResponse struct {
	Data struct {
		Code int    `json:"code"`
		Info CCData `json:"info"`
	} `json:"data"`
}

func NewCCGenerator(apikey string) *CCGenerator {
	generator := &CCGenerator{
		apikey:     apikey,
		baseURL:    "https://api.darkxcode.site/other/cc-generator/V1.5/",
		resultsDir: "result",
		client:     &http.Client{Timeout: 30 * time.Second},
	}
	generator.createResultsDir()
	return generator
}

func (c *CCGenerator) createResultsDir() {
	if _, err := os.Stat(c.resultsDir); os.IsNotExist(err) {
		os.Mkdir(c.resultsDir, 0755)
	}
}

func (c *CCGenerator) generateFilename() string {
	return "result.txt"
}

func (c *CCGenerator) saveToFile(data CCData, filename string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	filepath := c.resultsDir + "/" + filename
	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	saveData := fmt.Sprintf("%s|%s|%s|%s|%s",
		data.CC, data.Month, data.Year, data.CVV, data.Scheme)

	_, err = file.WriteString(saveData + "\n")
	return err
}

func (c *CCGenerator) generateCC(count int, ccType string, binNumber string) (*APIResponse, error) {
	req, err := http.NewRequest("GET", c.baseURL, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("submit", "1")
	q.Add("count", strconv.Itoa(count))
	q.Add("type", ccType)
	q.Add("apikey", c.apikey)
	if ccType == "CUSTOM" && binNumber != "" {
		q.Add("BIN", binNumber)
	}
	req.URL.RawQuery = q.Encode()

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	time.Sleep(time.Duration(rand.Intn(400)+100) * time.Millisecond)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResponse APIResponse
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return nil, err
	}

	return &apiResponse, nil
}

func (c *CCGenerator) processSingleRequest(index, totalCount int, ccType, binNumber string) bool {
	result, err := c.generateCC(1, ccType, binNumber)
	if err != nil {
		fmt.Printf("[%d/%d] ERROR: %s\n", index, totalCount, err.Error())
		return false
	}

	if result.Data.Code == 200 {
		data := result.Data.Info
		currentTime := time.Now().Format("2006-01-02 15:04:05")
		output := fmt.Sprintf("[%d/%d][%s] SUCCESS => %s|%s|%s|%s|%s | BY DARKXCODE V1.5",
			index, totalCount, currentTime, data.CC, data.Month, data.Year, data.CVV, data.Scheme)
		fmt.Println(output)

		filename := c.generateFilename()
		err := c.saveToFile(data, filename)
		if err != nil {
			fmt.Printf("Error saving to file: %s\n", err.Error())
		}
		return true
	} else {
		currentTime := time.Now().Format("2006-01-02 15:04:05")
		output := fmt.Sprintf("[%d/%d][%s] FAILED => %s | BY DARKXCODE V1.5",
			index, totalCount, currentTime, "UNKNOWN_ERROR")
		fmt.Println(output)
		return false
	}
}

func loadAPIKey() string {
	if _, err := os.Stat("settings.ini"); os.IsNotExist(err) {
		content := "[SETTINGS]\nAPIKEY = PASTE YOUR APIKEY HERE"
		ioutil.WriteFile("settings.ini", []byte(content), 0644)
		fmt.Println("[!] File settings.ini created. Please add your API key first!")
		os.Exit(0)
	}

	file, err := os.Open("settings.ini")
	if err != nil {
		fmt.Println("[!] Error reading settings.ini file!")
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "APIKEY = ") {
			apikey := strings.TrimPrefix(line, "APIKEY = ")
			if apikey == "PASTE YOUR APIKEY HERE" || strings.TrimSpace(apikey) == "" {
				fmt.Println("[!] Please update your API key in settings.ini file!")
				os.Exit(1)
			}
			return apikey
		}
	}

	fmt.Println("[!] APIKEY not found in settings.ini!")
	os.Exit(1)
	return ""
}

func getCardType() string {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("        [!] TYPE [!]")
	fmt.Println("1. VISA         2. MASTERCARD")
	fmt.Println("3. JCB          4. AMEX")
	fmt.Println("5. DISCOVER     6. RANDOM")
	fmt.Println("7. CUSTOM       99. EXIT")
	fmt.Println(strings.Repeat("=", 50))

	for {
		fmt.Print("[+] Choose number >> ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			fmt.Println("[!] Please enter a choice!")
			continue
		}

		choice, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("[!] Please enter a valid number!")
			continue
		}

		typeMapping := map[int]string{
			1:  "VISA",
			2:  "MASTERCARD",
			3:  "JCB",
			4:  "AMEX",
			5:  "DISCOVER",
			6:  "RANDOM",
			7:  "CUSTOM",
			99: "EXIT",
		}

		if cardType, exists := typeMapping[choice]; exists {
			if choice == 99 {
				fmt.Println("[+] Goodbye!")
				os.Exit(0)
			}
			return cardType
		} else {
			fmt.Println("[!] Invalid choice! Please choose 1-7 or 99 to exit.")
		}
	}
}

func getBINNumber() string {
	for {
		fmt.Print("[+] Please input BIN >> ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			fmt.Println("[!] BIN cannot be empty!")
			continue
		}

		// Remove non-digit characters
		var binBuilder strings.Builder
		for _, char := range input {
			if char >= '0' && char <= '9' {
				binBuilder.WriteRune(char)
			}
		}
		binInput := binBuilder.String()

		if len(binInput) < 6 {
			fmt.Println("[!] BIN must be at least 6 digits!")
			continue
		}

		return binInput
	}
}

func getThreads() int {
	for {
		fmt.Print("[+] Please input threads (min 1 & max 10) >> ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			fmt.Println("[!] Threads cannot be empty!")
			continue
		}

		threads, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("[!] Please enter a valid number!")
			continue
		}

		if threads >= 1 && threads <= 10 {
			return threads
		} else {
			fmt.Println("[!] Threads must be between 1 and 10!")
		}
	}
}

func getCount() int {
	for {
		fmt.Print("[+] Please input count (min 1 & max 1000) >> ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			fmt.Println("[!] Count cannot be empty!")
			continue
		}

		count, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("[!] Please enter a valid number!")
			continue
		}

		if count >= 1 && count <= 1000 {
			return count
		} else {
			fmt.Println("[!] Count must be between 1 and 1000!")
		}
	}
}

func main() {
	apikey := loadAPIKey()

	fmt.Printf("\n%s\n", strings.Repeat("=", 60))
	fmt.Println("                 DARKXCODE CC GENERATOR V1.5")
	fmt.Printf("%s\n", strings.Repeat("=", 60))

	cardType := getCardType()

	binNumber := ""
	if cardType == "CUSTOM" {
		binNumber = getBINNumber()
	}

	count := getCount()
	threads := getThreads()

	generator := NewCCGenerator(apikey)

	fmt.Printf("\n%s\n", strings.Repeat("=", 60))
	fmt.Println("GENERATION SETTINGS:")
	fmt.Printf("%s\n", strings.Repeat("=", 60))
	fmt.Printf("Type: %s\n", cardType)
	if cardType == "CUSTOM" {
		fmt.Printf("BIN: %s\n", binNumber)
	}
	fmt.Printf("Count: %d\n", count)
	fmt.Printf("Threads: %d\n", threads)
	fmt.Printf("%s\n\n", strings.Repeat("=", 60))

	fmt.Print("[+] Press Enter to start or 'n' to cancel: ")
	reader := bufio.NewReader(os.Stdin)
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(confirm)
	if confirm == "n" {
		fmt.Println("[+] Operation cancelled!")
		return
	}

	startTime := time.Now()
	successfulCount := 0

	var wg sync.WaitGroup
	taskChan := make(chan int, count)
	results := make(chan bool, count)

	// Start workers
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range taskChan {
				success := generator.processSingleRequest(index, count, cardType, binNumber)
				results <- success
			}
		}()
	}

	// Send tasks
	for i := 1; i <= count; i++ {
		taskChan <- i
		time.Sleep(100 * time.Millisecond)
	}
	close(taskChan)

	go func() {
		wg.Wait()
		close(results)
	}()

	for success := range results {
		if success {
			successfulCount++
		}
	}

	endTime := time.Now()
	totalTime := endTime.Sub(startTime).Seconds()

	fmt.Printf("\n%s\n", strings.Repeat("=", 60))
	fmt.Println("GENERATION COMPLETED!")
	fmt.Printf("%s\n", strings.Repeat("=", 60))
	fmt.Printf("Total requests: %d\n", count)
	fmt.Printf("Successful: %d\n", successfulCount)
	fmt.Printf("Failed: %d\n", count-successfulCount)
	fmt.Printf("Success rate: %.2f%%\n", float64(successfulCount)/float64(count)*100)
	fmt.Printf("Time taken: %.2f seconds\n", totalTime)
	fmt.Printf("Results saved in: %s/\n", generator.resultsDir)
	fmt.Printf("%s\n", strings.Repeat("=", 60))
}
