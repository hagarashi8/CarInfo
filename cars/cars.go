package cars

import "effMob/api"

// Модель машины из базы данных
type Car struct {
	ID int `gorm:"primaryKey" json:"id"`
	RegNum string  `json:"regNum"`
	Mark   string  `json:"mark"`
	Model  string  `json:"model"`
	Year   int32 `json:"year,omitempty"`
	OwnerName       string `json:"name"`
	OwnerSurname    string `json:"surname"`
	OwnerPatronymic string `json:"patronymic,omitempty"`
}

// Конвертирует машины, которые мы получаем с API, 
// в машины, которые мы грузим в базу данных и используем в коде
func NewFromAPI(car api.Car) *Car {
	return &Car{
		RegNum: car.RegNum,
		Mark: car.Mark,
		Model: car.Model,
		Year: car.Year,
		OwnerName: car.Owner.Name,
		OwnerSurname: car.Owner.Surname,
		OwnerPatronymic: car.Owner.Patronymic,
	}
}
