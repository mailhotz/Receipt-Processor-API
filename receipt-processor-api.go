package main

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Item struct {
	ShortDescription string          `json:"shortDescription" binding:"required"`
	Price            string          `json:"price" binding:"required"`
	priceAsDecimal   decimal.Decimal `json:"-"`
}

type Receipt struct {
	id             string
	Retailer       string          `json:"retailer" binding:"required"`
	PurchaseDate   string          `json:"purchaseDate" binding:"required"`
	PurchaseTime   string          `json:"purchaseTime" binding:"required"`
	Total          string          `json:"total" binding:"required"`
	totalAsDecimal decimal.Decimal `json:"-"`
	Items          []Item          `json:"items" binding:"required"`
}

type Result struct {
	Alphanumeric        string   `json:"alphanumeric"`
	RoundTotal          string   `json:"roundTotal"`
	MultipleTotal       string   `json:"multipleTotal"`
	NumberOfItems       string   `json:"numberOfItems"`
	DescriptionMultiple []string `json:"descriptionMultiple"`
	Llm                 string   `json:"LLM"`
	OddDay              string   `json:"oddDay"`
	PurchaseTime        string   `json:"purchaseTime"`
	Result              int      `json:"result"`
}

var Receipts []Receipt
var DebugObject Result

func CreateReceipt(c *gin.Context) {
	var receipt Receipt

	//Invalid json request
	if err := c.BindJSON(&receipt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	//Validate monetary values
	if err := ValidateTotal(&receipt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Total is not a valid value"})
		return
	}
	for i := 0; i < len(receipt.Items); i++ {
		if err := ValidatePrice(&receipt.Items[i]); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Price:" + receipt.Items[i].Price + " is not a valid value"})
		}
	}

	//Note: If this was going into production I'd validate dates & times as well, but not going to for this quick API test

	receipt.id = uuid.New().String()
	Receipts = append(Receipts, receipt)

	c.JSON(http.StatusOK, gin.H{"id": receipt.id})
}

func ValidateTotal(receipt *Receipt) error {
	val, err := decimal.NewFromString(receipt.Total)
	if err != nil {
		return err
	}
	receipt.totalAsDecimal = val
	return nil
}

func ValidatePrice(item *Item) error {
	val, err := decimal.NewFromString(item.Price)
	if err != nil {
		return err
	}

	item.priceAsDecimal = val
	return nil
}

// Made to confirm create endpoint is working correctly
func ListReceipts(c *gin.Context) {
	c.JSON(http.StatusOK, Receipts)
}

func GetPoints(c *gin.Context) {
	receipt := FindReceipt(c.Param("id"))

	err := binding.Validator.ValidateStruct(receipt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ID:'" + c.Param("id") + "' not found"})
		return
	}

	var result = CalculatePoints(&receipt, false)
	c.JSON(http.StatusOK, gin.H{"points": result})
}

// Prints step by step the score -- Used for debugging
func GetPointsWithSteps(c *gin.Context) {
	receipt := FindReceipt(c.Param("id"))

	err := binding.Validator.ValidateStruct(receipt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ID:'" + c.Param("id") + "' not found"})
		return
	}

	var result = CalculatePoints(&receipt, true)
	DebugObject.Result = result
	c.JSON(http.StatusOK, DebugObject)
}

func FindReceipt(id string) Receipt {
	var receipt Receipt
	for _, a := range Receipts {
		if a.id == id {
			receipt = a
			break
		}
	}

	return receipt
}

func CalculatePoints(receipt *Receipt, isStepThrough bool) int {
	result := 0

	//One point for every alphanumeric
	result += CalculateAlphanumeric(receipt.Retailer, isStepThrough)

	//50 points if total is round dollar amount
	result += CalculateIsRoundTotal(receipt.Total, isStepThrough)

	//25 points if total is a multipe of .25
	result += CalculateIsMultipleOfTwentyFive(receipt.Total, isStepThrough)

	//5 points for every 2 items of receipt
	result += CalculateNumberOfItems(len(receipt.Items), isStepThrough)

	//Trimmed length of item description is multiple of 3 multiply by 0.2 and round to nearest int
	result += CalculateItemDescriptionPoints(receipt.Items, isStepThrough)

	//LLM Step
	if isStepThrough {
		DebugObject.Llm = "I did not write this with an LLM so skip :)"
	}

	//6 points if the day is odd on purchase
	result += CalculateIsOddDay(receipt.PurchaseDate, isStepThrough)

	//10 points if purchases between 2pm and 4pm
	result += CalculateIsPurchaseTimeForPoint(receipt.PurchaseTime, isStepThrough)

	return result
}

func CalculateAlphanumeric(retailer string, isStepThrough bool) int {
	regex := regexp.MustCompile(`\w|\d+`)
	var result = len(regex.FindAllString(retailer, -1))
	if isStepThrough {
		DebugObject.Alphanumeric = "Regex to match alphanumeric characters added a score of " + strconv.Itoa(result)
	}

	return result
}

func CalculateIsRoundTotal(total string, isStepThrough bool) int {
	match_round, err_round := regexp.MatchString(`\d+\.00`, total)
	if err_round == nil && match_round {
		if isStepThrough {
			DebugObject.RoundTotal = "Total is a rounded dollar, adding 50 points"
		}

		return 50
	}

	return 0
}

func CalculateIsMultipleOfTwentyFive(total string, isStepThrough bool) int {
	match_multiple, err_multiple := regexp.MatchString(`\d+\.(00|25|50|75)`, total)
	if err_multiple == nil && match_multiple {
		if isStepThrough {
			DebugObject.MultipleTotal = "Total is a multiple of .25, adding 25 points"
		}

		return 25
	}

	return 0
}

func CalculateNumberOfItems(numOfItems int, isStepThrough bool) int {
	var itemMultiplier = (numOfItems / 2) * 5
	if isStepThrough {
		DebugObject.NumberOfItems = "There are " + strconv.Itoa(numOfItems) + " items. Adding " + strconv.Itoa(itemMultiplier) + " points"
	}

	return itemMultiplier
}

func CalculateItemDescriptionPoints(items []Item, isStepThrough bool) int {
	result := 0
	for _, i := range items {
		var trimmedDescription = strings.TrimSpace(i.ShortDescription)

		if len(trimmedDescription)%3 == 0 {
			var intToAdd = decimal.Decimal.Mul(i.priceAsDecimal, decimal.NewFromFloat(0.2)).RoundUp(0).IntPart()
			result += int(intToAdd)

			if isStepThrough {
				DebugObject.DescriptionMultiple = append(DebugObject.DescriptionMultiple, trimmedDescription+" is multiple of three, adding "+strconv.Itoa(int(intToAdd))+" points")
			}
		}
	}

	return result
}

func CalculateIsOddDay(date string, isStepThrough bool) int {
	regex_for_day := regexp.MustCompile(`-(\d{2}|\d{1})$`)
	var day = strings.Trim(string(regex_for_day.Find([]byte(date))), "-")
	var day_int, err = strconv.Atoi(day)
	if day_int%2 == 1 && err == nil {
		if isStepThrough {
			DebugObject.OddDay = "Day is odd, adding 6 points"
		}

		return 6
	}

	return 0
}

func CalculateIsPurchaseTimeForPoint(time string, isStepThrough bool) int {
	match_time, err_time := regexp.MatchString(`(14|15):\d{2}`, time)
	if err_time == nil && match_time {
		if isStepThrough {
			DebugObject.PurchaseTime = "Purchase was between 2pm and 4pm, adding 10 points"
		}

		return 10
	}

	return 0
}

func main() {
	r := gin.Default()
	//Don't allow extra fields on the JSON request
	binding.EnableDecoderDisallowUnknownFields = true

	r.GET("/receipts/:id/point", GetPoints)
	r.GET("receipts/:id/point/steps", GetPointsWithSteps)
	r.GET("/receipts", ListReceipts)
	r.POST("/receipt/process", CreateReceipt)

	r.Run() // listen and serve on 0.0.0.0:8080
}
