package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/biter777/countries"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nyaruka/phonenumbers"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

var client *whatsmeow.Client
var container *sqlstore.Container
var botDB *sql.DB
var isFirstRun = true

var htmlUI = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<style>
body{background:#0f172a;display:flex;justify-content:center;align-items:center;height:100vh;margin:0;font-family:system-ui,-apple-system,sans-serif;color:#f8fafc;}
.box{background:rgba(30,41,59,0.7);backdrop-filter:blur(10px);padding:2rem;border-radius:1rem;box-shadow:0 25px 50px -12px rgba(0,0,0,0.5);width:100%;max-width:24rem;border:1px solid rgba(255,255,255,0.1);text-align:center;}
input{width:100%;padding:0.75rem;margin-bottom:1rem;border-radius:0.5rem;border:1px solid #334155;background:#0f172a;color:#f8fafc;font-size:1rem;box-sizing:border-box;outline:none;}
input:focus{border-color:#3b82f6;}
button{width:100%;padding:0.75rem;border-radius:0.5rem;border:none;background:#3b82f6;color:white;font-size:1rem;font-weight:600;cursor:pointer;transition:all 0.2s;}
button:hover{background:#2563eb;}
.copy-btn{background:#10b981;margin-top:0.5rem;}
.copy-btn:hover{background:#059669;}
#res{display:none;margin-top:1.5rem;padding:1rem;background:#0f172a;border-radius:0.5rem;border:1px solid #334155;}
#codeTxt{display:block;font-size:1.5rem;letter-spacing:0.2em;font-weight:bold;margin-bottom:1rem;}
.loader{display:none;margin-top:1rem;color:#94a3b8;font-size:0.875rem;}
</style>
</head>
<body>
<div class="box">
<input type="text" id="phone" placeholder="Enter Phone Number" autocomplete="off">
<button id="btn" onclick="pair()">Connect</button>
<div id="loader" class="loader">Processing...</div>
<div id="res">
<span id="codeTxt"></span>
<button class="copy-btn" onclick="copy()">Copy</button>
</div>
</div>
<script>
async function pair(){
const p=document.getElementById('phone').value.replace(/\D/g,'');
if(!p)return;
document.getElementById('btn').style.display='none';
document.getElementById('loader').style.display='block';
document.getElementById('res').style.display='none';
try{
const r=await fetch('/link/pair/'+p);
const d=await r.json();
if(d.code){
document.getElementById('codeTxt').innerText=d.code;
document.getElementById('res').style.display='block';
}else{
alert(d.error||'Failed');
}
}catch(e){alert('Error connecting');}
document.getElementById('btn').style.display='block';
document.getElementById('loader').style.display='none';
}
function copy(){
const c=document.getElementById('codeTxt').innerText;
navigator.clipboard.writeText(c);
const cb=document.querySelector('.copy-btn');
cb.innerText='Copied!';
setTimeout(()=>cb.innerText='Copy',2000);
}
</script>
</body>
</html>`

func initSQLiteDB() {
	dbPath := "/data/kami_bot.db?_foreign_keys=on"
	if _, err := os.Stat("/data"); os.IsNotExist(err) {
		dbPath = "./kami_bot.db?_foreign_keys=on"
	}
	var err error
	botDB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		panic(err)
	}
	botDB.Exec(`CREATE TABLE IF NOT EXISTS sent_otps (msg_id TEXT PRIMARY KEY, sent_at DATETIME DEFAULT CURRENT_TIMESTAMP);`)
	botDB.Exec(`CREATE TABLE IF NOT EXISTS active_channels (channel_id TEXT PRIMARY KEY);`)
}

func isAlreadySent(id string) bool {
	var exists bool
	botDB.QueryRow(`SELECT EXISTS(SELECT 1 FROM sent_otps WHERE msg_id = ?)`, id).Scan(&exists)
	return exists
}

func markAsSent(id string) {
	botDB.Exec(`INSERT OR IGNORE INTO sent_otps (msg_id) VALUES (?)`, id)
}

func addActiveChannel(id string) bool {
	_, err := botDB.Exec(`INSERT OR IGNORE INTO active_channels (channel_id) VALUES (?)`, id)
	return err == nil
}

func removeActiveChannel(id string) bool {
	res, _ := botDB.Exec(`DELETE FROM active_channels WHERE channel_id = ?`, id)
	rows, _ := res.RowsAffected()
	return rows > 0
}

func getActiveChannels() []string {
	rows, err := botDB.Query(`SELECT channel_id FROM active_channels`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var channels []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		channels = append(channels, id)
	}
	return channels
}

func extractOTP(msg string) string {
	msg = regexp.MustCompile(`(\d)n([^\d\s])`).ReplaceAllString(msg, "$1 $2")
	msg = strings.ReplaceAll(msg, "nDont", " Dont")
	msg = strings.ReplaceAll(msg, "nDo ", " Do ")
	msg = strings.ReplaceAll(msg, "nYour", " Your")
	re := regexp.MustCompile(`\d[\d\-\s]{2,7}\d`)
	matches := re.FindAllString(msg, -1)
	for _, m := range matches {
		pureDigits := strings.ReplaceAll(strings.ReplaceAll(m, "-", ""), " ", "")
		if len(pureDigits) >= 4 && len(pureDigits) <= 8 {
			return m
		}
	}
	return "N/A"
}

func maskPhoneNumber(phone string) string {
	if len(phone) < 6 {
		return phone
	}
	return fmt.Sprintf("%s•••%s", phone[:3], phone[len(phone)-4:])
}

func getCountryFromPhone(phone string) string {
	if !strings.HasPrefix(phone, "+") {
		phone = "+" + phone
	}
	num, err := phonenumbers.Parse(phone, "")
	if err != nil {
		return "Unknown"
	}
	region := phonenumbers.GetRegionCodeForNumber(num)
	c := countries.ByName(region)
	if c != countries.Unknown {
		parts := strings.Fields(strings.Split(c.Info().Name, "-")[0])
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return region
}


// یہ فنکشن اب API کو کال کرنے کے بجائے ڈائریکٹ getSMSData() کو کال کر رہا ہے
func checkIvaOTPs(cli *whatsmeow.Client) {
	data, err := getSMSData()
	if err != nil || len(data) == 0 {
		return
	}

	channels := getActiveChannels()
	for i := len(data) - 1; i >= 0; i-- {
		row := data[i]
		if len(row) < 4 {
			continue
		}
		service := row[0]
		phone := row[1]
		fullMsg := html.UnescapeString(row[2])
		rawTime := row[3]
		msgID := fmt.Sprintf("IVA_%v_%v", phone, rawTime)
		
		if isAlreadySent(msgID) {
			continue
		}
		if isFirstRun {
			markAsSent(msgID)
			continue
		}
		
		countryName := getCountryFromPhone(phone)
        cFlag, _ := GetCountryWithFlag(countryName)
		flatMsg := strings.ReplaceAll(strings.ReplaceAll(fullMsg, "\n", " "), "\r", "")
		otpCode := extractOTP(flatMsg)
		maskedPhone := maskPhoneNumber(phone)
		
		messageBody := fmt.Sprintf("✨ *%s | %s Message* ⚡\n\n> *Time:* %s\n> *Country:* %s %s\n   *Number:* *%s*\n> *Service:* %s\n   *OTP:* *%s*\n\n> *Join For Numbers:* \n> ¹ https://chat.whatsapp.com/JqerYdpQZyY09LmX6WqFws?mode=gi_t\n*Full Message:*\n%s\n\n> © Developed by Nothing Is Impossible", cFlag, strings.ToUpper(service), rawTime, cFlag, countryName, maskedPhone, service, otpCode, flatMsg)
		
		for _, jidStr := range channels {
			jid, err := types.ParseJID(jidStr)
			if err == nil {
				cli.SendMessage(context.Background(), jid, &waProto.Message{
					Conversation: proto.String(strings.TrimSpace(messageBody)),
				})
			}
		}
		markAsSent(msgID)
	}
	isFirstRun = false
}

// یہ فنکشن بھی اب HTTP ریکویسٹ کی جگہ getNumbersData() کو کال کر رہا ہے
func handleNumbersCommand(cli *whatsmeow.Client, chatJID types.JID) {
	cli.SendMessage(context.Background(), chatJID, &waProto.Message{
		Conversation: proto.String("⏳ *Fetching numbers from server, please wait...*"),
	})
	
	apiData, err := getNumbersData()
	if err != nil || len(apiData.Numbers) == 0 {
		cli.SendMessage(context.Background(), chatJID, &waProto.Message{
			Conversation: proto.String("❌ *Error fetching numbers or no numbers found!*"),
		})
		return
	}

	groups := make(map[string][]string)
	for _, n := range apiData.Numbers {
		net := strings.ReplaceAll(n.Network, " ", "_")
		groups[net] = append(groups[net], n.Number)
	}

	for netName, nums := range groups {
		fileContent := strings.Join(nums, "\n")
		fileName := fmt.Sprintf("%s_Numbers.txt", netName)
		fileBytes := []byte(fileContent)
		uploaded, err := cli.Upload(context.Background(), fileBytes, whatsmeow.MediaDocument)
		if err != nil {
			continue
		}
		msg := &waProto.Message{
			DocumentMessage: &waProto.DocumentMessage{
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(fileBytes))),
				Mimetype:      proto.String("text/plain"),
				Title:         proto.String(fileName),
				FileName:      proto.String(fileName),
				Caption:       proto.String(fmt.Sprintf("📁 *%s*\n🔢 Total Numbers: %d", netName, len(nums))),
			},
		}
		cli.SendMessage(context.Background(), chatJID, msg)
	}
}

func formatJID(input string) string {
	input = strings.TrimSpace(input)
	if !strings.Contains(input, "@") {
		if len(input) > 15 {
			return input + "@newsletter"
		}
		return input + "@s.whatsapp.net"
	}
	return input
}

func handleCommands(evt *events.Message) {
	msgText := ""
	if evt.Message.GetConversation() != "" {
		msgText = evt.Message.GetConversation()
	} else if evt.Message.ExtendedTextMessage != nil {
		msgText = evt.Message.ExtendedTextMessage.GetText()
	}
	parts := strings.Fields(msgText)
	if len(parts) == 0 {
		return
	}
	cmd := strings.ToLower(parts[0])
	chatJID := evt.Info.Chat
	switch cmd {
	case ".menu":
		menuTxt := "━━━━━━━━━━━━━━━━━━━━━━\n🌟 *OTP BOT VIP MENU* 🌟\n━━━━━━━━━━━━━━━━━━━━━━\n\n🛠️ *User Commands:*\n▸ *.id* _(Get your ID or Group/Channel ID)_\n\n▸ *.numbers* _(Download available numbers line-by-line in TXT files)_\n\n⚙️ *Admin Commands:*\n▸ *.active <channel_id>* _(Activate OTP forwarding to a channel/chat)_\n\n▸ *.deactive <channel_id>* _(Stop OTP forwarding to a channel/chat)_\n\n━━━━━━━━━━━━━━━━━━━━━━\n© _Developed by Nothing Is Impossible_"
		client.SendMessage(context.Background(), chatJID, &waProto.Message{
			Conversation: proto.String(menuTxt),
		})
	case ".id":
		senderJID := evt.Info.Sender.ToNonAD().String()
		response := fmt.Sprintf("👤 *User ID:*\n`%s`\n\n📍 *Chat/Group ID:*\n`%s`", senderJID, chatJID.ToNonAD().String())
		if evt.Message.ExtendedTextMessage != nil && evt.Message.ExtendedTextMessage.ContextInfo != nil {
			quotedID := evt.Message.ExtendedTextMessage.ContextInfo.Participant
			if quotedID != nil {
				cleanQuoted := strings.Split(*quotedID, "@")[0] + "@" + strings.Split(*quotedID, "@")[1]
				cleanQuoted = strings.Split(cleanQuoted, ":")[0]
				response += fmt.Sprintf("\n\n↩️ *Replied ID:*\n`%s`", cleanQuoted)
			}
		}
		client.SendMessage(context.Background(), chatJID, &waProto.Message{
			Conversation: proto.String(response),
		})
	case ".numbers":
		handleNumbersCommand(client, chatJID)
	case ".active":
		if len(parts) < 2 {
			client.SendMessage(context.Background(), chatJID, &waProto.Message{
				Conversation: proto.String("⚠️ *Usage:* .active <id_or_number>"),
			})
			return
		}
		targetJID := formatJID(parts[1])
		if addActiveChannel(targetJID) {
			client.SendMessage(context.Background(), chatJID, &waProto.Message{
				Conversation: proto.String(fmt.Sprintf("✅ *Successfully Activated!*\nTarget: `%s`", targetJID)),
			})
		} else {
			client.SendMessage(context.Background(), chatJID, &waProto.Message{
				Conversation: proto.String("⚠️ *Already active or invalid ID.*"),
			})
		}
	case ".deactive":
		if len(parts) < 2 {
			client.SendMessage(context.Background(), chatJID, &waProto.Message{
				Conversation: proto.String("⚠️ *Usage:* .deactive <id_or_number>"),
			})
			return
		}
		targetJID := formatJID(parts[1])
		if removeActiveChannel(targetJID) {
			client.SendMessage(context.Background(), chatJID, &waProto.Message{
				Conversation: proto.String(fmt.Sprintf("🚫 *Successfully Deactivated!*\nTarget: `%s`", targetJID)),
			})
		} else {
			client.SendMessage(context.Background(), chatJID, &waProto.Message{
				Conversation: proto.String("⚠️ *Channel/Chat not found in active list.*"),
			})
		}
	}
}

func handler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		if !v.Info.IsFromMe {
			handleCommands(v)
		}
	case *events.LoggedOut:
		fmt.Println("Logout")
	case *events.Disconnected:
		fmt.Println("Disconnected")
	case *events.Connected:
		fmt.Println("Connected")
	}
}

func handlePairAPI(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, `{"error":"Invalid URL format"}`, 400)
		return
	}
	number := strings.TrimSpace(parts[3])
	number = strings.ReplaceAll(number, "+", "")
	number = strings.ReplaceAll(number, " ", "")
	number = strings.ReplaceAll(number, "-", "")
	if client != nil && client.IsConnected() {
		client.Disconnect()
		time.Sleep(2 * time.Second)
	}
	newDevice := container.NewDevice()
	tempClient := whatsmeow.NewClient(newDevice, waLog.Stdout("Pairing", "ERROR", true))
	tempClient.AddEventHandler(handler)
	err := tempClient.Connect()
	if err != nil {
		http.Error(w, `{"error":"Connection failed"}`, 500)
		return
	}
	code, err := tempClient.PairPhone(context.Background(), number, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		tempClient.Disconnect()
		http.Error(w, `{"error":"Pairing failed"}`, 500)
		return
	}
	go func() {
		for i := 0; i < 60; i++ {
			time.Sleep(1 * time.Second)
			if tempClient.Store.ID != nil {
				client = tempClient
				return
			}
		}
		tempClient.Disconnect()
	}()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"success": "true", "code": code, "number": number})
}

func handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	if client != nil && client.IsConnected() {
		client.Disconnect()
	}
	devices, _ := container.GetAllDevices(context.Background())
	for _, device := range devices {
		device.Delete(context.Background())
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"success": "true"})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	// آپ کے API والے روٹس جو api.go فائل کو لنک کر رہے ہیں
	http.HandleFunc("/api/sms", handleSMS)
	http.HandleFunc("/api/numbers", handleNumbers)

	// بوٹ کا ویب پینل اور پیئرنگ والے روٹس
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(htmlUI))
	})
	http.HandleFunc("/link/pair/", handlePairAPI)
	http.HandleFunc("/link/delete", handleDeleteSession)

	fmt.Println("سرور اور بوٹ پورٹ " + port + " پر چل رہے ہیں...")
	go func() {
		http.ListenAndServe("0.0.0.0:"+port, nil)
	}()

	// ڈیٹابیس اور واٹس ایپ کلائنٹ انیشلائز کرنا
	initSQLiteDB()
	dbPath := "/data/kami_bot.db?_foreign_keys=on"
	if _, err := os.Stat("/data"); os.IsNotExist(err) {
		dbPath = "file:./kami_bot.db?_foreign_keys=on"
	} else {
		dbPath = "file:" + dbPath
	}
	var err error
	container, err = sqlstore.New(context.Background(), "sqlite3", dbPath, waLog.Stdout("Database", "ERROR", true))
	if err == nil {
		deviceStore, err := container.GetFirstDevice(context.Background())
		if err == nil {
			client = whatsmeow.NewClient(deviceStore, waLog.Stdout("Client", "ERROR", true))
			client.AddEventHandler(handler)
			if client.Store.ID != nil {
				client.Connect()
			}
		}
	}

	// بیک گراؤنڈ میں OTP چیک کرنا
	go func() {
		for {
			if client != nil && client.IsLoggedIn() {
				checkIvaOTPs(client)
			}
			time.Sleep(5 * time.Second)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	if client != nil {
		client.Disconnect()
	}
}
