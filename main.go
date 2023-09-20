package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// Result representa los resultados del escaneo.
type Result struct {
	Endpoints []string `json:"endpoints"`
	HrefURLs  []string `json:"href_urls"`
}

// ScrapeData realiza una solicitud GET a la URL proporcionada y devuelve los datos como JSON.
func ScrapeData(url string) (Result, error) {
	// Anexa la ruta wp-json a la URL
	url = strings.TrimRight(url, "/") + "/wp-json"
	// Realiza una solicitud GET a la URL
	resp, err := http.Get(url)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	// Lee la respuesta como JSON
	var data map[string]interface{}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Result{}, err
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return Result{}, err
	}

	// Inicializa las listas de resultados
	var results Result
	results.Endpoints = make([]string, 0)
	results.HrefURLs = make([]string, 0)

	// Función recursiva para recorrer los datos y extraer los endpoints y las URL de href
	var traverse func(interface{}, string)
	traverse = func(d interface{}, path string) {
		switch v := d.(type) {
		case map[string]interface{}:
			for key, value := range v {
				if key == "href" {
					results.HrefURLs = append(results.HrefURLs, value.(string))
				} else {
					traverse(value, path+"/"+key)
				}
			}
		case []interface{}:
			for _, item := range v {
				traverse(item, path)
			}
		default:
			if path != "" {
				results.Endpoints = append(results.Endpoints, path)
			}
		}
	}

	// Llama a la función recursiva en la respuesta con una ruta vacía
	traverse(data, "")

	return results, nil
}

// SortResults ordena los resultados por endpoints y href URLs.
func SortResults(results []Result) []Result {
	for i := range results {
		results[i].Endpoints = sortStrings(results[i].Endpoints)
		results[i].HrefURLs = sortStrings(results[i].HrefURLs)
	}
	return results
}

// sortStrings ordena una lista de cadenas.
func sortStrings(strings []string) []string {
	for i := 0; i < len(strings); i++ {
		for j := i + 1; j < len(strings); j++ {
			if strings[i] > strings[j] {
				strings[i], strings[j] = strings[j], strings[i]
			}
		}
	}
	return strings
}

// WriteResults escribe los resultados en archivos.
func WriteResults(results []Result) error {
	for i, result := range results {
		endpointsFileName := fmt.Sprintf("endpoints_%d.txt", i)
		hrefURLsFileName := fmt.Sprintf("href_urls_%d.txt", i)

		endpointsFile, err := os.Create(endpointsFileName)
		if err != nil {
			return err
		}
		defer endpointsFile.Close()

		hrefURLsFile, err := os.Create(hrefURLsFileName)
		if err != nil {
			return err
		}
		defer hrefURLsFile.Close()

		for _, endpoint := range result.Endpoints {
			fmt.Fprintln(endpointsFile, endpoint)
		}

		for _, hrefURL := range result.HrefURLs {
			fmt.Fprintln(hrefURLsFile, hrefURL)
		}
	}
	return nil
}

func main() {
	// Define los flags de línea de comandos
	urlPtr := flag.String("u", "", "La URL del sitio de WordPress")
	filePtr := flag.String("f", "", "El archivo que contiene una lista de URLs de sitios de WordPress")
	flag.Parse()

	// Comprueba si se proporcionó una URL como entrada
	if *urlPtr != "" {
		// Raspa los datos de la URL
		data, err := ScrapeData(*urlPtr)
		if err != nil {
			fmt.Println("Error al raspar los datos:", err)
			return
		}

		// Ordena los datos por endpoints y href URLs
		sortedData := SortResults([]Result{data})

		// Escribe los datos ordenados en archivos
		err = WriteResults(sortedData)
		if err != nil {
			fmt.Println("Error al escribir los resultados:", err)
		}
	} else if *filePtr != "" {
		// Comprueba si se proporcionó un archivo como entrada
		file, err := os.Open(*filePtr)
		if err != nil {
			fmt.Println("Error al abrir el archivo:", err)
			return
		}
		defer file.Close()

		// Inicializa una lista vacía para almacenar los datos de todas las URL
		var data []Result

		// Escanea cada línea en el archivo
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			url := scanner.Text()
			scrapedData, err := ScrapeData(url)
			if err != nil {
				fmt.Printf("Error al raspar los datos de %s: %v\n", url, err)
			} else {
				data = append(data, scrapedData)
			}
		}

		// Ordena los datos por endpoints y href URLs
		sortedData := SortResults(data)

		// Escribe los datos ordenados en archivos
		err = WriteResults(sortedData)
		if err != nil {
			fmt.Println("Error al escribir los resultados:", err)
		}
	} else {
		fmt.Println("Debes proporcionar una URL (-u) o un archivo (-f) como entrada.")
	}
}
