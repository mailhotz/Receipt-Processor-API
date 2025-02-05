package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Item struct {
	ShortDescription string `json:"shortDescription" binding:"required"`
	Price            string `json:"price" binding:"required"`
}

type Receipt struct {
	id           string
	Retailer     string `json:"retailer" binding:"required"`
	PurchaseDate string `json:"purchaseDate" binding:"required"`
	PurchaseTime string `json:"purchaseTime" binding:"required"`
	Total        string `json:"total" binding:"required"`
	Items        []Item `json:"items" binding:"required"`
}

var Receipts []Receipt

//var validate *validator.Validate

func CreateReceipt(c *gin.Context) {
	var receipt Receipt

	//Invalid json to parse
	if err := c.BindJSON(&receipt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	receipt.id = uuid.New().String()
	Receipts = append(Receipts, receipt)

	c.JSON(http.StatusOK, gin.H{"id": receipt.id})
}

// Made to confirm create endpoint is working correctly
func ListReceipts(c *gin.Context) {
	c.JSON(http.StatusOK, Receipts)
}

func GetPoints(c *gin.Context) {

}

func main() {
	//validate = validator.New(validator.WithRequiredStructEnabled())
	r := gin.Default()

	r.GET("/receipts/:id/point", GetPoints)
	r.GET("/receipts/", ListReceipts)
	r.POST("/receipt/process", CreateReceipt)

	r.Run() // listen and serve on 0.0.0.0:8080
}
