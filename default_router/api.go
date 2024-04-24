package defaultrouter

import (
	"effMob/api"
	"effMob/cars"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type CarFilters struct {
	ID *int `gorm:"primaryKey" json:"id"`
	RegNum *string  `json:"regNum"`
	Mark   *string  `json:"mark"`
	Model  *string  `json:"model"`
	Year   *int32 `json:"year,omitempty"`
	OwnerName       *string `json:"name"`
	OwnerSurname    *string `json:"surname"`
	OwnerPatronymic *string `json:"patronymic,omitempty"`
}

// @Router /car [get]
// @Summary Выдаёт машины по фильтрам
// @Param page query int false "Номер страницы"
// @Param page_size query int false "Размер страницы"
// @Param id query int false "ID Машины"
// @Param model query string false "Модель Машины"
// @Param mark query string false "Марка Машины"
// @Param regnum query string false "Регистрационный номер"
// @Param owner_name query string false "Имя владельца"
// @Param owner_surname query string false "Фамилия владельца"
// @Param owner_patronymic query string false "Отчество владельца"
// @Success 200 {array} cars.Car "OK"
// @Success 404 "Машина не найдена"
// @Failure 500 "Внутренняя ошибка сервера"
func GetCarHandler(client *api.APIClient, db *gorm.DB, log *logrus.Logger) (http.HandlerFunc) {
	return func(w http.ResponseWriter, r *http.Request) {
		log := log.WithContext(r.Context())
		q := r.URL.Query()
		pageSize := 10;
		if q.Has("page_size") {
			log.Debugf("Размер страницы в запросе: %s\n", q.Get("page_size"))
			var err error
			pageSize, err = strconv.Atoi(q.Get("page_size"))
			if err != nil {
				log.WithError(err).Debugln("Размер страницы не является числом, используем размер 10 строк")
				pageSize = 10
			}
		}
		log.Debugf("Итоговый размер страницы: %d\n", pageSize)
		page:= 1;
		if q.Has("page") {
			log.Debugf("Страница в запросе: %s\n", q.Get("page"))
			var err error
			page, err = strconv.Atoi(q.Get("page"))
			if err != nil {
				log.WithError(err).Debugln("Страница не является числом, используем первую страниц")
				page = 1
			}
		}
		log.Debugf("Итоговая страница: %d\n", page)
		result := make([]cars.Car, pageSize)
		car := carFromQuery(r.URL.Query())
		log.Traceln("Создан объект машины из параметров запроса")

		tx := db.Where(car).Offset((page-1)*pageSize).Limit(pageSize).Find(&result)
		if tx.Error != nil {
			log.WithContext(r.Context())
			log.WithError(tx.Error)
			log.Debugln("Ошибка получения от базы данных")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Traceln("Найдены машины")
		resjs, err := json.Marshal(result)
		if err != nil {
			log.WithError(err).Error("Не удалось сериализовать объект")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(resjs)
	}
}

func carFromQuery(q url.Values) (car *CarFilters){
	car = new(CarFilters)
	if q.Has("mark"){
		mark := q.Get("mark")
		car.Mark = &mark
	}
	if q.Has("year"){
		year, err := strconv.Atoi(q.Get("year"))
		if err == nil {
			y := int32(year)
			car.Year = &y
		}
	}
	if q.Has("model"){ 
		m := q.Get("model")
		car.Model = &m
	}
	if q.Has("regnum"){ 
		r := q.Get("regnum") 
		car.RegNum = &r
	}
	if q.Has("owner_name"){ 
		on := q.Get("owner_name")
		car.OwnerName = &on
	}
	if q.Has("owner_surname"){ 
		osu := q.Get("owner_surname")
		car.OwnerSurname = &osu
	}
	if q.Has("owner_patronymic"){ 
		op := q.Get("owner_patronymic")
		car.OwnerPatronymic = &op
	}
	return
}

// @Param regnum body defaultrouter.AddCarHandler.RequestBody true "Номера машин"
// @Summary Добавляет машины в базу данных
// @Description Принимает массив автомобильных гос номеров и добавляет соответствующие машины в базу данных сервиса
// @Success 200 {array} cars.Car
// @Failure 400 "Номер машины не был принят сторонним API/Тело запроса не соответствовало структуре"
// @Failure 500 "Внутренняя ощибка сервера"
// @Failure 502 "Внутренняя ошибка на стороннем API"
// @Router /car [post]
func AddCarHandler(client *api.APIClient, db *gorm.DB, log *logrus.Logger) (http.HandlerFunc) {
	type RequestBody struct{
		RegNums []string `json:"regNums"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		log := log.WithContext(r.Context())
		var body RequestBody
		req, err := io.ReadAll(r.Body)
		if err != nil {
			log.WithError(err).Errorln("Ошибка при чтении тела запроса")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		log.Debugln(string(req))
		err = json.Unmarshal(req, &body)
		if err != nil {
			log.WithError(err).Debug("Ошибка парсинга тела запроса")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		log.Traceln("JSON спарсен, результат: ", body)
		res := make([]cars.Car, 0) 
		for _, regNum := range body.RegNums {
			car, resp, err := client.DefaultApi.InfoGet(r.Context(), regNum)
			if err != nil {
				log.WithError(err).Errorln("Ошибка получения машины со стороннего API")
				if resp.StatusCode == http.StatusBadRequest{
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(fmt.Sprintf("Неправильный регистрационный номер: %s\n", regNum)))
				} else {
					w.WriteHeader(http.StatusBadGateway)
					w.Write([]byte("502 Bad Gateway"))
				}
				return
			}
			carconv := cars.NewFromAPI(car)
			tx := db.Create(carconv)
			if tx.Error != nil {
				log.WithError(tx.Error).Errorln("Не удалось создать машину")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("500 Internal Server Error"))
				return
			}
			res = append(res, *carconv)
		}
		resbytes, _:= json.Marshal(res)
		w.Write(resbytes)
	}
}

// @Summary Удаляет машину по ID
// @Param id query int true "ID Машины"
// @Success 200 {object} cars.Car
// @Failure 400 "Не был указан ID или был указан неправильный"
// @Failure 404 "Машина с указаным ID не была найдена"
// @Failure 500 "Внутренняя ошибка сервера"
// @Router /car [delete]
func DeleteCarHandler(client *api.APIClient, db *gorm.DB, log *logrus.Logger) (http.HandlerFunc) {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if !q.Has("id") {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Необходимо указать ID"))
			log.Debugln("Не был указан ID в запросе:",q.Encode())
			return
		}
		log.Tracef("В запросе был ID: %s\n", q.Get("id"))
		id, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Неправильный ID"))
			log.WithError(err)
			log.Debugln("Ошибка конвертации в число при удалении, параметры: ",q.Encode())
			return
		}
		log.Traceln("ID - Целое число")
		car := cars.Car{ID: id}
		tx := db.Where(&car).First(&car)
		if tx.Error != nil {
			w.WriteHeader(http.StatusNotFound)
			log.WithError(tx.Error)
			return
		}
		log.Traceln("Успешно найдена машина")
		tx = db.Delete(&car)
		if tx.Error != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.WithError(tx.Error)
			return
		}
		log.Traceln("Успешно удалена машина")
		carbytes, err := json.Marshal(&car)
		w.WriteHeader(http.StatusOK)
		w.Write(carbytes)
	}
}

// @Summary Изменяет существующую машину
// @Param patch body defaultrouter.PutCarHandler.CarPatch true "Изменения машин. ID обязателен."
// @Success 200 {array} defaultrouter.PutCarHandler.CarPatch "Было изменено больше 0 машин. Возращает не принятые изменения"
// @Failure 400 "Тело запроса - не валидный запрос"
// @Failure 404 "Не было изменено ни одной машины"
// @Failure 500 "Внутренняя ошибка сервера"
// @Router /car [put]
func PutCarHandler(client *api.APIClient, db *gorm.DB, log *logrus.Logger) (http.HandlerFunc) {
	type CarPatch struct {
		ID *int `gorm:"primaryKey" json:"id"`
		RegNum *string  `json:"regNum"`
		Mark   *string  `json:"mark"`
		Model  *string  `json:"model"`
		Year   *int32 `json:"year,omitempty"`
		OwnerName       *string `json:"name"`
		OwnerSurname    *string `json:"surname"`
		OwnerPatronymic *string `json:"patronymic,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var patches []CarPatch
		req, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 Internal Server Error"))
			log.WithError(err)
			log.Errorln("Ошибка при чтении тела запроса")
			return
		}
		log.Traceln("Тело запроса прочитано")
		err = json.Unmarshal(req, &patches)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("400 Bad Request"))
			log.WithError(err)
			log.Debugln("Ошибка при парсинге тела запроса")
			log.Debugln(req)
			return
		}
		log.Traceln("Парсинг тела удался")
		notFounds := make([]CarPatch, 0, len(patches))
		for _, car := range patches {
			if car.ID == nil {
				log.Debugln()
				continue
			}
			var dbcar cars.Car
			tx := db.Where(&car).First(&dbcar)
			if tx.Error != nil {
				notFounds = append(notFounds, car)
				continue
			}
			db.Model(&dbcar).Updates(&car)
		}
		if len(notFounds) != len(patches) {
			w.WriteHeader(http.StatusOK)
			b, _ := json.Marshal(&notFounds)
			w.Write(b)
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404 Not found"))
		}
	}
}
