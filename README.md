# nCore Auto Seed Checker

This is a GO application designed as an automatic seed checker for the nCore.pro website.

## User Guide

1. **URL Verification:** After downloading, verify the URLs in the `main.go` file:
   - `loginUrl`: this should point to `login.php`.
   - `activityUrl`: this should point to `hitnrun.php`.

2. **Filling in Login Information:** Provide your username and password using command line flags:
   - `-u your_username`
   - `-p your_password`

3. **Setting the Output Directory:**
   - Specify the directory using the `-o` flag.
   - **Note:** This application **does not re-download the torrents**, you need to set your torrent client to download the torrent files into the same folder, where they will be automatically added to the torrent client.

4. **Building the Application:**
   - Required dependencies:
      - `go get github.com/chromedp/chromedp`
      - `go get github.com/chromedp/cdproto/cdp`
   - On Windows: `go build -o ncore_automation.exe .`
   - On Linux/Mac: `go build -o ncore_automation .`
   - Run the above command in the directory where the downloaded file is located.

5. **Running the Application:**
   - **One-time Run:** 
     `./ncore_automation -u user -p pass -o ./torrents`
   - **Web Interface:** 
     `./ncore_automation -u user -p pass -o ./torrents -web`
     Open [http://localhost:8080](http://localhost:8080) to view history.
     You can change the port with `-port 9090`.
   - **Continuous Mode (Every 10 min):**
     `./ncore_automation -u user -p pass -o ./torrents -web -interval 10m`

   Use the `-d` flag to enable logging to `log.txt`.

---

**Important:** I am not responsible for any illegal content distribution. This application only automates manual steps.

HUN
# nCore Auto Seed Checker

Ez egy nCore.pro oldalhoz készült automatikus seed checker GO alkalmazás.

## Használati útmutató

1. **URL ellenőrzés:** Miután letöltötted, ellenőrizd a `main.go` fájlban:
   - `loginUrl`: ennek a `login.php`-ra kell vezetnie.
   - `activityUrl`: ennek a `hitnrun.php`-ra kell vezetnie.
   
2. **Login adatok megadása:** Add meg a felhasználónevedet és jelszavadat parancssori kapcsolókkal:
   - `-u felhasznalonev`
   - `-p jelszo`

3. **Kimeneti könyvtár beállítása:**
   - Az `-o` kapcsolóval adhatod meg, hova kerüljenek a letöltött `.torrent` fájlok.
   - **Figyelem:** Ez az alkalmazás **nem tölti vissza a torrentet**, hanem egy tetszőleges torrent alkalmazásba kell beállítani, hogy ugyanabba a mappába töltse le a torrent fájlokat, ahol automatikusan hozzáadja őket a torrent alkalmazáshoz.

4. **Alkalmazás buildelése:**
   - Szükséges kiegészítők:
      - go get github.com/chromedp/chromedp
      - go get github.com/chromedp/cdproto/cdp 
   - Windows rendszeren: `go build -o ncore_automation.exe .`
   - Linux/Mac rendszeren: `go build -o ncore_automation .`
   - Futtasd le a fenti parancsot a letöltött fájl mappájában.

5. **Alkalmazás futtatása:**
   - **Egyszeri futtatás:**
     `./ncore_automation -u user -p pass -o ./torrents`
   - **Web felület:** 
     `./ncore_automation -u user -p pass -o ./torrents -web`
     Megtekintheted a [http://localhost:8080](http://localhost:8080) címen.
     A portot a `-port 9090` kapcsolóval módosíthatod.
   - **Folyamatos futtatás (10 percenként):**
     `./ncore_automation -u user -p pass -o ./torrents -web -interval 10m`

`-d` kapcsolóval log.txt-be logol

---

**Fontos:** Semmilyen felelősséget nem vállalok jogsértő tartalom terjesztéséért, ez az alkalmazás csak a manuális lépéseket váltja ki.
