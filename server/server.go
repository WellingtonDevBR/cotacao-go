package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Cotacao struct {
	Bid string `json:"bid"`
}

func getCotacao(ctx context.Context) (Cotacao, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return Cotacao{}, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return Cotacao{}, err
	}
	defer resp.Body.Close()

	var result map[string]Cotacao
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Cotacao{}, err
	}

	return result["USDBRL"], nil
}

func fetchCotacaoWithRetry(retries int) (Cotacao, error) {
	var cotacao Cotacao
	var err error

	for i := 0; i < retries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		cotacao, err = getCotacao(ctx)
		if err == nil {
			return cotacao, nil
		}
		log.Println("Erro ao obter cotação, tentando novamente:", err)
		time.Sleep(50 * time.Millisecond) // Pequeno intervalo entre tentativas
	}

	return Cotacao{}, err
}

func saveCotacao(ctx context.Context, db *sql.DB, cotacao Cotacao) error {
	query := "INSERT INTO cotacoes (bid, timestamp) VALUES (?, ?)"
	_, err := db.ExecContext(ctx, query, cotacao.Bid, time.Now().Unix())
	return err
}

func cotacaoHandler(w http.ResponseWriter, r *http.Request) {
	cotacao, err := fetchCotacaoWithRetry(3)
	if err != nil {
		http.Error(w, "Erro ao obter cotação", http.StatusInternalServerError)
		log.Println("Erro ao obter cotação após múltiplas tentativas:", err)
		return
	}

	db, err := sql.Open("sqlite3", "./cotacoes.db")
	if err != nil {
		http.Error(w, "Erro ao abrir banco de dados", http.StatusInternalServerError)
		log.Println("Erro ao abrir banco de dados:", err)
		return
	}
	defer db.Close()

	ctxDB, cancelDB := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancelDB()

	if err := saveCotacao(ctxDB, db, cotacao); err != nil {
		http.Error(w, "Erro ao salvar cotação", http.StatusInternalServerError)
		log.Println("Erro ao salvar cotação:", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cotacao)
}

func main() {
	db, err := sql.Open("sqlite3", "./cotacoes.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS cotacoes (id INTEGER PRIMARY KEY, bid TEXT, timestamp INTEGER)")
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/cotacao", cotacaoHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
