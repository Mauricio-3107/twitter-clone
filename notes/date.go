package notes

import (
	"time"
)

// 1) Parsing here
// layout := "2006-01-02" // The layout for your date "1996-07-31"
// dobStr := "1996-07-31"

// dob, err := time.Parse(layout, dobStr)
// if err != nil {
// 	// Handle the error if the parsing fails
// 	panic(err)
// }

// 2) Creating the date and not parsing here
// createdDate := time.Date(1963, time.July, 10, 23, 23, 0, 0, time.UTC)

// 3) Creating the date and not parsing here
func main() {
	layout := "2006-01-02"
	dateStr := "1990-07-10"
	createdDate, err := time.Parse(layout, dateStr)
	if err != nil {
		// Handle the error if the parsing fails
		panic(err)
	}
	// Set the time part to midnight (00:00:00)
	createdDate = createdDate.UTC().Truncate(24 * time.Hour)

	// General code
	// user := models.NewUser{
	// 	Name:     "Pollo Ramirez",
	// 	Dob:      createdDate,
	// 	Email:    "m@pollo.com",
	// 	Password: "qwerty",
	// }

	// u, err := us.Create(user)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(u)
}
