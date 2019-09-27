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
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type config struct {
	Token     string
	DBName    string
	Admins    []int
	Recaptcha recaptchaConfig `json:"recaptcha"`
}
type recaptchaConfig struct {
	V2         bool
	PrivateKey string
	PublicKey  string
	Domain     string
	Port       int
	MinScore   float32
}
type request struct {
	CaptchaCode int
	WantToken   string
}
type sPageIn struct {
	mux sync.Mutex //Nearly everywhere we are writing to PageIn. Also when reading, instantly we write to it
	//This is a variable to define what "Admins" are going to do; The key is the ID of the admin and the value is the page they want to do. Here is the list of the pages
	// 0: Nowhere but the main menu; Send the tokens for the link to start verification
	// 1: Admin whats to add a new text
	// 2: Admin whats to remove a token
	PageIn map[int]int
}
type sCaptchaToCheck struct {
	mux            sync.Mutex //We write to it, or instantly delete it after reading from it; So no need to RWMutex
	CaptchaToCheck map[int]request
}
type recaptchaResponse struct {
	Success     bool      `json:"success"`
	Score       float32   `json:"score"`
	Action      string    `json:"action"`
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
}

var bot *tgbotapi.BotAPI
var PageIn sPageIn
var CaptchaToCheck sCaptchaToCheck
var Config config
var ConfigFileName string

//1 is normal
//2 is recaptcha v2
//3 is recaptcha v3
var CaptchaMode = byte(1)

//Web stuff
const (
	pageHead = `<html><head>
	<style>.error{color:#ff0000;} div{margin: auto; text-align: center;} .ack{color:#0000ff;} p{text-align: center;}</style><title>Recaptcha Test</title></head>
<body><div style="width:100%"><div style="width: 50%;margin: 0 auto;">`
	pageTopV2 = `<p>Please check the dialog and choose OK after</p><form action="/" method="POST">
	    <script src="https://www.google.com/recaptcha/api.js"></script>
		<div style="" class="g-recaptcha" data-sitekey="%s"></div>
		<input style="display: none" name="chatid" type="text" value="%s">
		<input style="display: none" name="dbtoken" type="text" value="%s">
		<div><input type="submit" name="button" value="Ok"></div>
</form>`
	pageTopV3 = `<script src="https://www.google.com/recaptcha/api.js?render=%s"></script>
  	<script>
  	grecaptcha.ready(function() {
		grecaptcha.execute('%s', {action: 'homepage'}).then(function(token) {
			document.getElementById("token").value = token;
			document.getElementById("myForm").submit();
		});
	});
	</script>
	<p>Please wait...</p>
	<form id="myForm" action="/" method="POST">
	<input style="display: none" id="token" name="g-recaptcha-response" type="text">
	<input style="display: none" name="chatid" type="text" value="%s">
	<input style="display: none" name="dbtoken" type="text" value="%s">
</form>
	`
	pageBottom = `</div></div></body></html>`
	anError    = `<p class="error">%s</p>`
	anOK       = `<p class="ack">%s</p><script>
function Redirect() 
{  
	window.location="https://telegram.me/%s"; 
}
setTimeout('Redirect()', 1000);
</script>`
)
const recaptchaURLLocal = "http://%s:%d/?chatid=%d&dbtoken=%s"
const recaptchaServerName = "https://www.google.com/recaptcha/api/siteverify"
const Version = "1.1.2 / Build 6"

func init() {
	rand.Seed(time.Now().UnixNano()) //Make randoms, random
}

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

	{
		//At first read the config file
		confF, err := ioutil.ReadFile(ConfigFileName)
		if err != nil {
			panic("Cannot read the config file. (io Error) " + err.Error())
		}
		err = json.Unmarshal(confF, &Config)
		if err != nil {
			panic("Cannot read the config file. (Parse Error) " + err.Error())
		}
		//Load captcha settings
		if Config.Recaptcha.PublicKey != "" {
			if Config.Recaptcha.V2 {
				CaptchaMode = 2
			} else {
				CaptchaMode = 3
			}
		}
	}

	//If needed fire up the http server
	if CaptchaMode != 1 {
		http.HandleFunc("/", homePage)
		log.Println("Starting the web server on port", Config.Recaptcha.Port)
		go func() {
			if err := http.ListenAndServe(":"+strconv.FormatInt(int64(Config.Recaptcha.Port), 10), nil); err != nil {
				log.Fatal("failed to start server", err)
			}
		}()
	}

	//Load db
	err := LoadDB(Config.DBName)
	if err != nil {
		panic("Cannot access database: " + err.Error())
	}
	defer CloseDB()

	//Setup the bot
	bot, err = tgbotapi.NewBotAPI(Config.Token)
	if err != nil {
		panic("Cannot initialize the bot: " + err.Error())
	}

	//Initialize the Captcha and Page in
	CaptchaToCheck.CaptchaToCheck = make(map[int]request)
	PageIn.PageIn = make(map[int]int)

	log.Printf("Bot authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}
		//Check if message is command
		if update.Message.IsCommand() {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			switch update.Message.Command() {
			case "start":
				if strings.Contains(update.Message.Text, " ") { //Check if bot is lunched from deeplink
					token := strings.Split(update.Message.Text, " ")[1] //This gets the token
					go processToken(token, update.Message.From.ID, update.Message.Chat.ID)
					continue
				}
				if !checkInArray(update.Message.From.ID, Config.Admins) { //Check admin
					msg.Text = "Welcome! Please send the token you received to get the text or the link."
				} else {
					msg.Text = "Hello!\nYou are the admin of this bot.\nHere is a list of commands:\n\n/add : Use this command to add a link or text. This will later result in a \"token\". Share that token to users to let them receive the text or link.\n/remove : Remove a token\n/list : Lists all of the tokens and values\n/id : Get the ID of anyone that sends it to bot. Can be used to define new admins.\n/about : Just a about screen"
				}
			case "add":
				if !checkInArray(update.Message.From.ID, Config.Admins) { //Check admin
					log.Println("Unauthorized access from id", update.Message.From.ID, "and username", update.Message.From.UserName, "and name", update.Message.From.FirstName, update.Message.From.LastName)
					msg.Text = "You are not the admin of this bot!"
				} else { //User is admin
					PageIn.mux.Lock()
					PageIn.PageIn[update.Message.From.ID] = 1
					PageIn.mux.Unlock()
					msg.Text = "Please send a text or a link to create a token for it"
				}
			case "remove":
				if !checkInArray(update.Message.From.ID, Config.Admins) { //Check admin
					log.Println("Unauthorized access from id", update.Message.From.ID, "and username", update.Message.From.UserName, "and name", update.Message.From.FirstName, update.Message.From.LastName)
					msg.Text = "You are not the admin of this bot!"
				} else { //User is admin
					PageIn.mux.Lock()
					PageIn.PageIn[update.Message.From.ID] = 2
					PageIn.mux.Unlock()
					msg.Text = "Please send the token to remove it from database"
				}
			case "list":
				if !checkInArray(update.Message.From.ID, Config.Admins) { //Check admin
					log.Println("Unauthorized access from id", update.Message.From.ID, "and username", update.Message.From.UserName, "and name", update.Message.From.FirstName, update.Message.From.LastName)
					msg.Text = "You are not the admin of this bot!"
				} else { //User is admin
					go func(id int64) { //Gather all of the links
						msg := tgbotapi.NewMessage(id, "")
						list, err := ListAllValues()
						if err != nil {
							msg.Text = "Error getting the list: " + err.Error()
						} else {
							if len(list) == 0 {
								msg.Text = "The database is empty!"
							} else {
								var sb strings.Builder
								for k, v := range list {
									sb.WriteString("`")
									sb.WriteString(k)
									sb.WriteString("`")
									sb.WriteString(" : ")
									sb.WriteString(v)
									sb.WriteString("\n")
								}
								msg.Text = sb.String()
								msg.ParseMode = "markdown"
								msg.DisableWebPagePreview = true
							}
						}
						botSend(msg)
					}(update.Message.Chat.ID)
					continue
				}
			case "cancel":
				CaptchaToCheck.mux.Lock()
				delete(CaptchaToCheck.CaptchaToCheck, update.Message.From.ID)
				CaptchaToCheck.mux.Unlock()
				msg.Text = "You can now send a token to bot to access it's data."
				if checkInArray(update.Message.From.ID, Config.Admins) { //Check admin
					PageIn.mux.Lock()
					PageIn.PageIn[update.Message.From.ID] = 0 //Goto nowhere
					PageIn.mux.Unlock()
				}
			case "about":
				msg.Text = "Made by Hirbod Behnam\nGolang\nSource code at https://github.com/HirbodBehnam/CaptchaBot\nBackend version " + Version
			case "id": //Send the id to anyone
				msg.Text = strconv.FormatInt(int64(update.Message.From.ID), 10)
			default:
				msg.Text = "I don't know that command"
			}
			botSend(msg)
		} else {
			if checkInArray(update.Message.From.ID, Config.Admins) { //If user is admin...
				PageIn.mux.Lock()
				switch PageIn.PageIn[update.Message.From.ID] {
				case 1: //Admin wants to add a string or link
					PageIn.PageIn[update.Message.From.ID] = 0
					PageIn.mux.Unlock()
					token, err := InsertValue(update.Message.Text)
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					if err != nil {
						msg.Text = "Error in inserting this string in database: " + err.Error()
					} else {
						msg.Text = "Successfully created the text in database!\nThe key is `" + token + "` .\nAlso you can use this link to let the users start the bot directly:\nhttps://telegram.me/" + escapeMarkdown(bot.Self.UserName) + "?start=" + token + "\nShare it with users."
						msg.ParseMode = "markdown"
					}
					botSend(msg)
					continue //Continue to server other updates
				case 2: //Admin whats to delete a token
					PageIn.PageIn[update.Message.From.ID] = 0
					PageIn.mux.Unlock()
					err := RemoveKey(update.Message.Text)
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					if err != nil {
						msg.Text = "Error in deleting this token from database: " + err.Error()
					} else {
						msg.Text = "Successfully deleted token `" + update.Message.Text + "` from database."
						msg.ParseMode = "markdown"
					}
					botSend(msg)
					continue
				} //Otherwise admin way want to see a link
				PageIn.mux.Unlock()
			}
			//So basically we have 2 scenarios:
			// 1. The value passed to bot is only numbers: This means that the user is replying to a captcha
			// 2. The value is letters only: User is requesting a text or link. We shall send him a qr code
			if a, err := strconv.Atoi(update.Message.Text); err == nil { //Here we have scenario 1; Every thing is a number
				if CaptchaMode == 1 {
					go func(userEntry int, chatID int64, id int) {
						msg := tgbotapi.NewMessage(chatID, "")
						req := safeReadCaptchaToCheckAndDelete(id)
						if req.WantToken == "" {
							msg.Text = "Please send the bot a token first."
						} else if userEntry == req.CaptchaCode { //Captcha is ok
							str, err := ReadValue(req.WantToken)
							if err != nil {
								msg.Text = "Error retrieving data from database: " + err.Error()
							} else {
								msg.Text = str
							}
						} else {
							msg.Text = "Captcha fail. Please try again by sending the _token_ again."
							msg.ParseMode = "markdown"
						}
						botSend(msg)
					}(a, update.Message.Chat.ID, update.Message.From.ID)
				} else {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "The token you provided is in valid or does not exists.")
					botSend(msg)
				}
			} else { //Here we have scenario 2; At first try to read it from database
				go processToken(update.Message.Text, update.Message.From.ID, update.Message.Chat.ID)
			}
		}
	}
}

//Just handle errors here
func botSend(message tgbotapi.Chattable) {
	_, err := bot.Send(message)
	if err != nil {
		log.Println("Error on sending a message:", err.Error())
	}
}

//Generate the captcha
func processToken(token string, id int, chatID int64) { //This function will be always called with go
	if HasKey(token) {
		//Prepare the QR Code
		switch CaptchaMode {
		case 1: //Send a normal captcha
			digits := captcha.RandomDigits(8)
			{ //Convert digits to int to save 4 bits on every user :|
				numDigits := 0
				for i := 0; i < 8; i++ { //Build the number
					numDigits *= 10
					numDigits += int(digits[i])
				}
				CaptchaToCheck.mux.Lock()
				CaptchaToCheck.CaptchaToCheck[id] = request{numDigits, token}
				CaptchaToCheck.mux.Unlock()
			}
			qrImage := captcha.NewImage(strconv.FormatInt(int64(id), 10), digits, 200, 100)
			var buf bytes.Buffer
			if err := jpeg.Encode(&buf, qrImage.Paletted, nil); err != nil {
				msg := tgbotapi.NewMessage(chatID, "Error on encoding captcha.")
				log.Println("Error on encoding captcha.", err.Error())
				botSend(msg)
				return
			}
			file := tgbotapi.FileBytes{Bytes: buf.Bytes(), Name: strconv.FormatInt(int64(id), 10)}
			msg := tgbotapi.NewPhotoUpload(chatID, file)
			msg.Caption = "Please enter the number in this image\n/cancel to turn back"
			botSend(msg)
		case 2:
			msg := tgbotapi.NewMessage(chatID, "Open this url and complete the captcha:\n"+fmt.Sprintf(recaptchaURLLocal, Config.Recaptcha.Domain, Config.Recaptcha.Port, chatID, token))
			msg.DisableWebPagePreview = true
			botSend(msg)
		case 3:
			msg := tgbotapi.NewMessage(chatID, "Open this url and wait:\n"+fmt.Sprintf(recaptchaURLLocal, Config.Recaptcha.Domain, Config.Recaptcha.Port, chatID, token))
			msg.DisableWebPagePreview = true
			botSend(msg)
		}
	} else { //The link is broken
		msg := tgbotapi.NewMessage(chatID, "The token you provided is in valid or does not exists.")
		botSend(msg)
	}
}

//Check recaptcha from web post
func processRequest(request *http.Request) bool {
	recaptchaResponse := request.FormValue("g-recaptcha-response")
	result, err := checkRecaptcha("127.0.0.1", recaptchaResponse)
	if err != nil {
		log.Println("recaptcha server error", err)
	}
	if CaptchaMode == 2 {
		return result.Success
	} else {
		return result.Score >= Config.Recaptcha.MinScore
	}
}

//Load the page
func homePage(writer http.ResponseWriter, request *http.Request) {
	err := request.ParseForm() // Must be called before writing response
	id := request.FormValue("chatid")
	token := request.FormValue("dbtoken")
	fmt.Fprint(writer, pageHead)
	if err != nil {
		fmt.Fprintf(writer, fmt.Sprintf(anError, err))
	} else {
		_, buttonClicked := request.Form["g-recaptcha-response"]
		if buttonClicked {
			if processRequest(request) {
				a, _ := strconv.Atoi(id)
				fmt.Fprint(writer, fmt.Sprintf(anOK, "Sent the code via telegram!", bot.Self.UserName))
				go sendValueWithBot(int64(a), token)
			} else {
				if CaptchaMode == 2 {
					fmt.Fprintf(writer, fmt.Sprintf(anError, "Recaptcha was incorrect; try again."))
				} else {
					fmt.Fprintf(writer, fmt.Sprintf(anError, "Unfortunately you are not worthy enough to access this right now."))
				}
			}
		} else {
			if CaptchaMode == 2 {
				fmt.Fprint(writer, fmt.Sprintf(pageTopV2, Config.Recaptcha.PublicKey, id, token))
			} else {
				fmt.Fprint(writer, fmt.Sprintf(pageTopV3, Config.Recaptcha.PublicKey, Config.Recaptcha.PublicKey, id, token))
			}
		}
	}
	fmt.Fprint(writer, pageBottom)
}
func checkRecaptcha(remoteip, response string) (r recaptchaResponse, err error) {
	resp, err := http.PostForm(recaptchaServerName,
		url.Values{"secret": {Config.Recaptcha.PrivateKey}, "remoteip": {remoteip}, "response": {response}})
	if err != nil {
		log.Printf("Post error: %s\n", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Read error: could not read body: %s", err)
		return
	}
	err = json.Unmarshal(body, &r)
	if err != nil {
		log.Println("Read error: got invalid JSON: %s", err)
		return
	}
	return
}

//Gets a value from database and sends it to bot
func sendValueWithBot(id int64, token string) {
	value, err := ReadValue(token)
	msg := tgbotapi.NewMessage(id, "")
	if err == nil {
		msg.Text = value
	} else {
		msg.Text = "Error getting value from database: " + err.Error()
	}
	botSend(msg)
}

//With mutex, read the captcha from CaptchaToCheck and delete the value after
func safeReadCaptchaToCheckAndDelete(id int) request {
	CaptchaToCheck.mux.Lock()
	res := CaptchaToCheck.CaptchaToCheck[id]
	delete(CaptchaToCheck.CaptchaToCheck, id)
	CaptchaToCheck.mux.Unlock()
	return res
}

//A small function to check if an array contains a key
func checkInArray(value int, array []int) bool {
	for _, i := range array {
		if i == value {
			return true
		}
	}
	return false
}

//In case that anything have one of these chars it must be escaped when sending markdown
func escapeMarkdown(str string) string {
	chars := []string{"_", "*", "[", "`"}
	for _, k := range chars {
		str = strings.ReplaceAll(str, k, `\`+k)
	}
	return str
}
