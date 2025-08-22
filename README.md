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
   - On Windows: `go build -o ncore_automation.exe main.go`
   - On Linux: `go build -o ncore_automation main.go`
   - Run the above command in the directory where the downloaded file is located.

5. **Running the Application:**
   - The program can now be run or scheduled, depending on the user's choice. 😄

   Use the `-d` flag to enable logging to `log.txt`.

6. **Running with Docker:**
   - Build the Docker image:
     `docker build -t ncore-automation .`
   - Run the container with credentials and paths via environment variables:
     `docker run -e NCORE_USER=your_username -e NCORE_PASS=your_password -e OUTPUT_DIR=/downloads -e PORT=8080 -p 8080:8080 -v /host/path:/downloads ncore-automation`

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
   - Windows rendszeren: `go build -o ncore_automation.exe main.go`
   - Linux rendszeren: `go build -o ncore_automation main.go`
   - Futtasd le a fenti parancsot a letöltött fájl mappájában.

6. **Alkalmazás futtatása:**
   - A program ezután már futtatható vagy ütemezhető, ez már a felhasználó választása. 😄

-d kapcsolóval log.txt-be logol

7. **Docker használata:**
   - Docker image buildelése:
     `docker build -t ncore-automation .`
   - Konténer futtatása környezeti változókkal:
     `docker run -e NCORE_USER=felhasznalonev -e NCORE_PASS=jelszo -e OUTPUT_DIR=/downloads -e PORT=8080 -p 8080:8080 -v /host/path:/downloads ncore-automation`

---

**Fontos:** Semmilyen felelősséget nem vállalok jogsértő tartalom terjesztéséért, ez az alkalmazás csak a manuális lépéseket váltja ki.
