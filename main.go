package main

import (
	"context"
	"fmt"
	"math/rand"
	CRand "crypto/rand"
	"time"
	"log"
	"unicode"
	"strings"
	"math/big"
	"os"
	"flag"
	"regexp"
	"github.com/chromedp/chromedp"
	petname "github.com/dustinkirkland/golang-petname"
	"accountCreateUmbrella/Mail"
	"accountCreateUmbrella/DnsSRV"
)

type Account struct {
	Mail   string
	Password string
	ApiKey string
}
var (
	lower    = "abcdefghijklmnopqrstuvwxyz"
	upper    = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits   = "0123456789"
	specials = "!@#$%&*"
	all      = lower + upper + digits + specials
	re = regexp.MustCompile(`(?m)(https:\/\/signup\.umbrella\.com\/password\/[^\s=]+(?:=[^\s=]*)*)`)
)




func capitalize(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func generateNumberStarting() string {
	rand.Seed(time.Now().UnixNano())

	// Génère les 6 derniers chiffres (000000 à 999999)
	suffix := rand.Intn(1_000_000)

	// Assemble le numéro complet au format string
	return fmt.Sprintf("298%06d", suffix)
}

func main() {

         
	var outputFile = flag.String("o", "UmbrellaApiKey.txt", "Chemin du fichier de sortie")
	flag.Parse()

	go DnsSRV.StartDns()
    store := Mail.NewMailList()

	server := Mail.InitServer("dns.wleberre.fr", store)
	go Mail.StartServer(server)
	DataAccount := Inscription()
	log.Println("[+] Signin with mail : "+DataAccount.Mail)
   	log.Println("[+] Wait a mail")

	out := false
	activationLink := ""
	for {
		time.Sleep(3 * time.Second)

		mails := store.GetAll()

		for _, m := range mails {
			parts := strings.SplitN(m.From , "@", 2)

			if(len(parts) == 2 && strings.ToLower(parts[1]) == "spmail.opendns.com"){
				out = true
				activationLink = re.FindString(m.Data)
				log.Println("[+] Found activation link : "+activationLink)
				break
			}
		}
		if out {
			break
		}
	}
	DataAccount = ActivationLink(DataAccount,activationLink)
	GetApiKey(DataAccount,*outputFile)
}

func Inscription() Account {
	// Activer le mode graphique (headless false)
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true), // Navigateur visible
		//chromedp.Flag("disable-gpu", false),    // Active l'accélération GPU
		//chromedp.Flag("start-maximized", true), // Maximise la fenêtre
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// Crée un contexte lié à l'onglet Chrome
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Timeout global
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// À adapter : URL réelle du formulaire
	formURL := "https://signup.umbrella.com/?return_to=https://dashboard.opendns.com/"
	rand.Seed(time.Now().UnixNano())
	firstname := petname.Generate(1, "-")
	lastname := petname.Generate(1, "-")
	comapagny := petname.Generate(1, "-")

	err := chromedp.Run(ctx,
		chromedp.Navigate(formURL),

		// Attendre que le champ username apparaisse
		chromedp.WaitVisible(`#firstName`, chromedp.ByID),

		// Remplir les champs
		chromedp.SendKeys(`#firstName`, capitalize(firstname), chromedp.ByID),
		chromedp.SendKeys(`#lastName`, capitalize(lastname), chromedp.ByID),
		chromedp.SendKeys(`#email`, firstname+"."+lastname+"@dns.wleberre.fr", chromedp.ByID),
		chromedp.SendKeys(`#phone`, generateNumberStarting(), chromedp.ByID),
		chromedp.SetValue(`#country`, "FR", chromedp.ByID),
		chromedp.SetValue(`#company`, comapagny, chromedp.ByID),
		chromedp.SetValue(`#employeeCount`, "1-50", chromedp.ByID),
		chromedp.Click(`#termInput`, chromedp.ByID),
		// Cliquer sur le bouton de soumission
		chromedp.Click(`button[type=submit]`, chromedp.ByQuery),
		// Attendre pour voir l'effet
		chromedp.Sleep(5*time.Second),
	)
	if err != nil {
		fmt.Println("Erreur :", err)
		return Account{}
	}

	log.Println("[+] Form submit")
	return Account{Mail:firstname+"."+lastname+"@dns.wleberre.fr"}
}



func ActivationLink(account Account,domain string) Account {
	pass := generatePassword(account.Mail)
	account.Password = pass
	// Activer le mode graphique (headless false)
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true), // Navigateur visible
		//chromedp.Flag("disable-gpu", false),    // Active l'accélération GPU
		//chromedp.Flag("start-maximized", true), // Maximise la fenêtre
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// Crée un contexte lié à l'onglet Chrome
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Timeout global
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// À adapter : URL réelle du formulaire

	err := chromedp.Run(ctx,
		chromedp.Navigate(domain),
		chromedp.WaitVisible(`#password`, chromedp.ByID),
		chromedp.SendKeys(`#password`, pass, chromedp.ByID),
		chromedp.SendKeys(`#password_confirmation`, pass, chromedp.ByID),
		chromedp.Click(`button[type=submit]`, chromedp.ByQuery),
		chromedp.Sleep(5*time.Second),
	)
	if err != nil {
		fmt.Println("Erreur :", err)
		return Account{}
	}

	log.Println("[+] Password set : "+pass)
	return account
}



func GetApiKey(account Account, file string) Account {
	user := account.Mail
	pass := account.Password

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 50*time.Second)
	defer cancel()
	var html string
	var htmlDump string
	// Séquence d'actions sur la page A (formulaire)
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://login.umbrella.com/"),
		chromedp.WaitVisible(`#username`, chromedp.ByID),
		chromedp.SendKeys(`#username`, user, chromedp.ByID),
		chromedp.SendKeys(`#password`, pass, chromedp.ByID),
		chromedp.Click(`button[type=submit]`, chromedp.ByQuery),
		//chromedp.Sleep(6*time.Second), // attendre que l'action se déclenche
		chromedp.WaitVisible(`#onboarding-wizard`, chromedp.ByID),
		//on skip le tuto
		chromedp.WaitVisible(`//a[span[contains(text(), "Skip Step")]]`, chromedp.BySearch),
		chromedp.Click(`//a[span[contains(text(), "Skip Step")]]`, chromedp.BySearch),
		chromedp.WaitVisible(`//a[span[contains(text(), "Skip This Step")]]`, chromedp.BySearch),
		chromedp.Click(`//a[span[contains(text(), "Skip This Step")]]`, chromedp.BySearch),
		chromedp.WaitVisible(`//button[span[contains(text(), "Start Using Cisco Umbrella")]]`, chromedp.BySearch),
		chromedp.Click(`//button[span[contains(text(), "Start Using Cisco Umbrella")]]`, chromedp.BySearch),
		chromedp.Sleep(3*time.Second),
		//on go sur le api key
		chromedp.Evaluate(`window.location.href = "#/investigate/tokens-view"`, nil),
		chromedp.Sleep(4*time.Second),
		chromedp.WaitVisible(`//*[@id="dashx-shim-content"]/div/div/div[2]/div/div/div[1]/a`, chromedp.BySearch),
		chromedp.Click(`//*[@id="dashx-shim-content"]/div/div/div[2]/div/div/div[1]/a`, chromedp.BySearch),
		chromedp.Sleep(2*time.Second),
		chromedp.SendKeys(`#new-token-title`, "siem", chromedp.ByID),
		chromedp.Click(`//*[@id="dashx-shim-content"]/div/div/div[2]/div/div/div[2]/div[3]/input`, chromedp.BySearch),
		chromedp.Sleep(3*time.Second), 
		chromedp.OuterHTML(".investigateApp", &html, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
	)
	if err != nil {
		fmt.Println(htmlDump)
		log.Println("[-] Get token error :", err)
		return Account{}
	}
	reg := regexp.MustCompile(`[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}`)
	fmt.Println(html)
	uuids := reg.FindAllString(html, -1)
	for _, uuid := range uuids {
		account.ApiKey = uuid
		log.Println("[+] Success API Key found : "+uuid)
		saveToFile(uuid,file)
		log.Println("[+] API Key stored in  : "+flag)
		return account
	}
		log.Println("[-] Token not found")
		return Account{}
}




func randomChar(charset string) byte {
	nBig,_ := CRand.Int(CRand.Reader, big.NewInt(int64(len(charset))))
	return charset[nBig.Int64()]
}

func generatePassword(username string) string {
	for {
		pass := []byte{
			randomChar(lower),
			randomChar(upper),
			randomChar(digits),
			randomChar(specials),
		}

		for len(pass) < 12 {
			pass = append(pass, randomChar(all))
		}

		password := string(pass)
		if !strings.Contains(strings.ToLower(password), strings.ToLower(username)) {
			return password
		}
	}
}
func saveToFile(content string, filename string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}
