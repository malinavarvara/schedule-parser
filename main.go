package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/gocolly/colly/v2"
	_ "github.com/mattn/go-sqlite3"
)

type Day struct {
	nameDay string
	Classes []Lesson
}

type Lesson struct {
	name    string
	time    string
	address string
	teacher string
	status  string
}

func insertData(db *sql.DB, days []Day) {
	insertSQL := `INSERT INTO users (name_day, name_subject, time, address, teacher) VALUES (?, ?, ?, ?, ?)`

	for _, day := range days {
		for _, lesson := range day.Classes {
			_, err := db.Exec(insertSQL, day.nameDay, lesson.name, lesson.time, lesson.address, lesson.teacher)
			if err != nil {
				log.Printf("Ошибка при вставке данных в базу: %v", err)
			} else {
				log.Printf("Запись добавлена: %s, %s, %s, %s, %s", day.nameDay, lesson.name, lesson.time, lesson.address, lesson.teacher)
			}
		}
	}
}

func main() {
	var days []Day
	c := colly.NewCollector(
		colly.AllowURLRevisit(),
		colly.AllowedDomains("timetable.spbu.ru"),
	)
	c.OnHTML(".panel", func(e *colly.HTMLElement) {
		d := Day{}
		var less []Lesson
		txt := e.ChildText(".panel-title")
		if txt != "" {
			d.nameDay = txt
		}
		e.ForEach(".common-list-item", func(index int, child *colly.HTMLElement) {
			l := Lesson{}
			name := child.ChildText(".col-sm-4")
			time := child.ChildText(".col-sm-2")
			address := child.ChildText(".address-modal-btn")
			teacher := child.ChildText("a")
			if name != "" {
				l.name = name
				l.time = time
				l.address = address
				l.teacher = teacher
			}
			less = append(less, l)
		})
		d.Classes = less
		days = append(days, d)

	})

	c.OnRequest(func(r *colly.Request) {
		// Создаем новые куки
		cookie := &http.Cookie{
			Name:  "_culture",
			Value: "ru",
		}
		// Устанавливаем куки в запрос
		r.Ctx.Put("cookie", cookie)
	})

	startURL := "https://timetable.spbu.ru/AMCP/StudentGroupEvents/Primary/394787/2024-10-28" // Начальная страница
	c.OnRequest(func(r *colly.Request) {
		cookie := &http.Cookie{
			Name:  "_culture",
			Value: "ru",
		}
		r.Headers.Add("Cookie", cookie.Name+"="+cookie.Value)
	})

	err := c.Visit(startURL)
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("sqlite3", "./timeTable.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Создаем таблицу, если её нет
	createTableSQL := `CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name_day TEXT NOT NULL,
		name_subject TEXT NOT NULL,
        time STRING,
		address STRING,
		teacher STRING

    );`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Не удалось создать таблицу: %v", err)
	}

	// Вставка данных из массива days в базу данных
	insertData(db, days)

	// Обработчик для ошибок
	c.OnError(func(_ *colly.Response, err error) {
		log.Println("Произошла ошибка:", err)
	})

	// Обработка завершения работы
	c.OnScraped(func(r *colly.Response) {

		fmt.Println("Сканирование завершено", r.Request.URL)
	})
}
