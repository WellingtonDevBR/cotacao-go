package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

type Cotacao struct {
	Bid string `json:"bid"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/cotacao", nil)
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Erro ao fazer requisição:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Erro na resposta do servidor: %s", resp.Status)
	}

	var cotacao Cotacao
	if err := json.NewDecoder(resp.Body).Decode(&cotacao); err != nil {
		log.Fatal("Erro ao decodificar resposta:", err)
	}

	cotacaoStr := "Dólar: " + cotacao.Bid
	file, err := os.Create("cotacao.txt")
	if err != nil {
		log.Fatal("Erro ao criar arquivo:", err)
	}
	defer file.Close()

	if _, err := file.WriteString(cotacaoStr); err != nil {
		log.Fatal("Erro ao escrever no arquivo:", err)
	}

	log.Println("Cotação salva em cotacao.txt:", cotacaoStr)
}
