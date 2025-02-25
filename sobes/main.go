package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// Константы
const (
	bufferSize   = 1000                                                    // Максимальный размер буфера
	flushTimeout = time.Minute                                             // Таймаут для периодического сброса буфера
	apiURL       = "https://development.kpi-drive.ru/_api/facts/save_fact" // URL API для сохранения
	authToken    = "Bearer 48ab34464a5573519725deb5865cc74c"               // Токен авторизации
)

// Глобальные переменные
var (
	buffer    []Fact                        // Буфер для хранения записей
	mutex     sync.Mutex                    // Мьютекс для защиты буфера
	flushChan = make(chan struct{})         // Канал для сигнала о сбросе буфера
	sendChan  = make(chan Fact, bufferSize) // Канал для последовательной отправки записей
)

// Fact — структура данных, соответствующая полям из ТЗ
type Fact struct {
	PeriodStart         string `json:"period_start,omitempty"`
	PeriodEnd           string `json:"period_end,omitempty"`
	PeriodKey           string `json:"period_key,omitempty"`
	IndicatorToMoID     int    `json:"indicator_to_mo_id,omitempty"`
	IndicatorToMoFactID int    `json:"indicator_to_mo_fact_id,omitempty"`
	Value               int    `json:"value,omitempty"`
	FactTime            string `json:"fact_time,omitempty"`
	IsPlan              int    `json:"is_plan,omitempty"`
	AuthUserID          int    `json:"auth_user_id,omitempty"`
	Comment             string `json:"comment,omitempty"`
}

func main() {
	// Запускаем горутину для последовательной отправки записей
	go sendWorker()

	// Запускаем горутину для периодического сброса буфера
	go bufferWorker()

	// Запускаем HTTP-сервер для приема записей
	http.HandleFunc("/submit", submitHandler)
	log.Println("Сервер был запущен.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// submitHandler — обработчик HTTP-запросов
func submitHandler(w http.ResponseWriter, r *http.Request) {
	var fact Fact
	if err := json.NewDecoder(r.Body).Decode(&fact); err != nil {
		// Обработка ошибки декодирования JSON
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Добавляем запись в буфер и проверяем его размер
	mutex.Lock()
	buffer = append(buffer, fact)
	if len(buffer) >= bufferSize {
		// Если буфер заполнен, отправляем сигнал на сброс
		flushChan <- struct{}{}
	}
	mutex.Unlock()

	// Отправляем успешный статус
	w.WriteHeader(http.StatusAccepted)

}

// bufferWorker — фоновая горутина для периодического или принудительного сброса буфера
func bufferWorker() {
	ticker := time.NewTicker(flushTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Сброс по таймеру
			flushBuffer()
		case <-flushChan:
			// Сброс по сигналу (буфер полон)
			flushBuffer()
		}
	}
}

// flushBuffer — переносит записи из буфера в канал для отправки
func flushBuffer() {
	mutex.Lock()
	bufferLen := len(buffer)
	mutex.Unlock()

	// Если буфер пуст, ничего не делаем
	if bufferLen == 0 {
		return
	}

	// Переносим записи из буфера в канал
	mutex.Lock()
	for len(buffer) > 0 {
		fact := buffer[0]
		buffer = buffer[1:]
		mutex.Unlock()
		sendChan <- fact
		mutex.Lock()
	}
	mutex.Unlock()
}

// sendWorker — горутина для последовательной отправки записей в API
func sendWorker() {
	for fact := range sendChan {
		// Отправляем каждую запись по очереди
		sendFact(fact)
	}
}

// sendFact — отправка одной записи в API
func sendFact(fact Fact) {
	// Формируем данные в формате urlencoded
	data := url.Values{}
	data.Set("period_start", fact.PeriodStart)
	data.Set("period_end", fact.PeriodEnd)
	data.Set("period_key", fact.PeriodKey)
	data.Set("indicator_to_mo_id", strconv.Itoa(fact.IndicatorToMoID))
	data.Set("indicator_to_mo_fact_id", strconv.Itoa(fact.IndicatorToMoFactID))
	data.Set("value", strconv.Itoa(fact.Value))
	data.Set("fact_time", fact.FactTime)
	data.Set("is_plan", strconv.Itoa(fact.IsPlan))
	data.Set("auth_user_id", strconv.Itoa(fact.AuthUserID))
	data.Set("comment", fact.Comment)

	// Создаём POST-запрос
	req, err := http.NewRequest("POST", apiURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		// Обработка ошибки создания запроса
		log.Printf("Ошибка создания запроса: %v", err)
		return
	}

	// Устанавливаем заголовки
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", authToken)

	// Отправляем запрос
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// Обработка ошибки отправки запроса
		log.Printf("Ошибка отправки запроса: %v", err)
		return
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		log.Printf("Неуспешный статус ответа: %s. Запись: %+v", resp.Status, fact)
		return
	}

	log.Println("Данные отправлены успешно")
}
