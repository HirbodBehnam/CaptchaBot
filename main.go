package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dchest/captcha"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

type config struct {
	Token  string
	DBName string
	Admins []int
}

type request struct {
	CaptchaCode int
	WantToken 	string
}

//This is a variable to define what "Admins" are going to do; The key is the ID of the admin and the value is the page they want to do. Here is the list of the pages
// 0: Nowhere but the main menu; Send the tokens for the link to start verification
// 1: Admin whats to add a new text
// 2: Admin whats to remove a token
var PageIn = make(map[int]int)
var CaptchaToCheck = make(map[int]request)
var Config config
var ConfigFileName string
const Version = "0.1.0 / Build 0"

func main() {
	{ //Parse arguments
		configFileName := flag.String("config", "config.json", "The config filename")
		help := flag.Bool("h", false, "Show help")
		flag.Parse()

		ConfigFileName = *configFileName

		if *help {
			fmt.Println("Created by Hirbod Behnam")
			fmt.Println("Source at https://github.com/HirbodBehnam/CaptchaBot")
			fmt.Println("Version", Version)
			flag.PrintDefaults()
			os.Exit(0)
		}
	}

	//At first read the config file
	confF, err := ioutil.ReadFile(ConfigFileName)
	if err != nil {
		panic("Cannot read the config file. (io Error) " + err.Error())
	}
	err = json.Unmarshal(confF, &Config)
	if err != nil {
		panic("Cannot read the config file. (Parse Error) " + err.Error())
	}

	//
	err = LoadDB(Config.DBName)
	if err != nil{
		panic("Cannot access database: " + err.Error())
	}
	//Setup the bot
	bot, err := tgbotapi.NewBotAPI(Config.Token)
	if err != nil {
		log.Fatal(err)
	}

	//bot.Debug = Verbose

	log.Printf("Bot authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}
		if update.Message.IsCommand(){
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			switch update.Message.Command() {
			case "start":
				if !checkInArray(update.Message.From.ID,Config.Admins) { //Check admin
					msg.Text = "Welcome! Please send the token you received to get the text or the link."
				}else{
					msg.Text = "Hello!\nYou are the admin of this bot.\nHere is a list of commands:\n\n/new : Use this command to add a link or text. This will later result in a \"token\". Share that token to users to let them receive the text or link.\n/remove : Remove a token\n/id : Get the ID of anyone that sends it to bot. Can be used to define new admins.\n/about : Just a about screen"
				}
			case "add":
				if !checkInArray(update.Message.From.ID,Config.Admins){ //Check admin
					log.Println("Unauthorized access from id",update.Message.From.ID,"and username",update.Message.From.UserName,"and name",update.Message.From.FirstName,update.Message.From.LastName)
					msg.Text = "You are not the admin of this bot!"
				}else{ //User is admin
					PageIn[update.Message.From.ID]=1
					msg.Text = "Please send a text or a link to create a token for it"
				}
			case "remove":
				if !checkInArray(update.Message.From.ID,Config.Admins){ //Check admin
					log.Println("Unauthorized access from id",update.Message.From.ID,"and username",update.Message.From.UserName,"and name",update.Message.From.FirstName,update.Message.From.LastName)
					msg.Text = "You are not the admin of this bot!"
				}else{ //User is admin
					PageIn[update.Message.From.ID] = 2
					msg.Text = "Please send the token to remove it from database"
				}
			case "cancel":
				delete(CaptchaToCheck,update.Message.From.ID)
				msg.Text = "You can now send a token to bot to access it's data."
				if checkInArray(update.Message.From.ID,Config.Admins){ //Check admin
					PageIn[update.Message.From.ID] = 0 //Goto nowhere
				}
			case "about":
				msg.Text = "Made by Hirbod Behnam\nGolang\nSource code at https://github.com/HirbodBehnam/CaptchaBot\nVersion "+ Version
			case "id": //Send the id to anyone
				msg.Text = strconv.FormatInt(int64(update.Message.From.ID),10)
			default:
				msg.Text = "I don't know that command"
			}
			bot.Send(msg)
		}else{
			if checkInArray(update.Message.From.ID,Config.Admins){ //If user is admin...
				switch PageIn[update.Message.From.ID] {
				case 1: //Admin wants to add a string or link
					PageIn[update.Message.From.ID] = 0
					token , err := InsertValue(update.Message.Text)
					msg := tgbotapi.NewMessage(update.Message.Chat.ID,"")
					if err != nil{
						msg.Text = "Error in inserting this string in database: " + err.Error()
					}else{
						msg.Text = "Successfully created the text in database!\nThe key is `" + token + "` . Share it with users."
						msg.ParseMode = "markdown"
					}
					bot.Send(msg)
					continue //Continue to server other updates
				case 2: //Admin whats to delete a token
					PageIn[update.Message.From.ID] = 0
					err := RemoveKey(update.Message.Text)
					msg := tgbotapi.NewMessage(update.Message.Chat.ID,"")
					if err != nil{
						msg.Text = "Error in deleting this token from database: " + err.Error()
					}else{
						msg.Text = "Successfully deleted token " + update.Message.Text + " from database."
					}
					bot.Send(msg)
					continue
				} //Otherwise admin way want to see a link
			}
			//So basically we have 2 scenarios:
			// 1. The value passed to bot is only numbers: This means that the user is replying to a captcha
			// 2. The value is letters only: User is requesting a text or link. We shall send him a qr code
			if userEntry, err := strconv.Atoi(update.Message.Text); err == nil { //Here we have scenario 1; Every thing is a number
				msg := tgbotapi.NewMessage(update.Message.Chat.ID,"")
				if userEntry == CaptchaToCheck[update.Message.From.ID].CaptchaCode{ //Captcha is ok
					str, err := ReadValue(CaptchaToCheck[update.Message.From.ID].WantToken)
					if err != nil{
						msg.Text = "Error retrieving data from database: " + err.Error()
					}else{
						msg.Text = str
					}
				}else{
					msg.Text = "Captcha fail. Please try again by sending the _token_ again."
					msg.ParseMode = "markdown"
				}
				delete(CaptchaToCheck,update.Message.From.ID) //Delete the captcha result; We dont want it anymore
				bot.Send(msg)
			}else{ //Here we have scenario 2; At first try to read it from database
				if HasKey(update.Message.Text){
					//Prepare the QR Code
					go func(message string,id int,chatID int64) {
						digits := captcha.RandomDigits(8)
						{ //Convert digits to int to save 4 bits on every user :|
							numDigits := 0
							for i := 0;i<8;i++{//Build the number
								numDigits *= 10
								numDigits += int(digits[i])
							}
							CaptchaToCheck[id] = request{numDigits,message}
						}
						qrImage := captcha.NewImage(strconv.FormatInt(int64(id),10),digits,200,100)
						var buf bytes.Buffer
						if err := jpeg.Encode(&buf, qrImage.Paletted,nil); err != nil {
							panic(err.Error())
						}
						file := tgbotapi.FileBytes{Bytes:buf.Bytes(),Name:strconv.FormatInt(int64(id),10)}
						msg := tgbotapi.NewPhotoUpload(chatID,file)
						msg.Caption = "Please enter the number in this image\n/cancel to turn back"
						bot.Send(msg)
					}(update.Message.Text, update.Message.From.ID, update.Message.Chat.ID)
				}else{ //The link is broken
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "The token you provided is in valid or does not exists.")
					_ , _ = bot.Send(msg)
				}
			}
		}
	}
}

//A small function to check if an array contains a key
func checkInArray(value int,array []int) bool {
	for _,i := range array{
		if i == value{
			return true
		}
	}
	return false
}
