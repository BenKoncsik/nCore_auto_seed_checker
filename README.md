# Ncore Auto Seed Checker

Ez egy Ncore oldalhoz készült automatikus seed checker GO alkalmazás.

## Használati útmutató

1. **URL ellenőrzés:** Miután letöltötted, ellenőrizd a `main.go` fájlban:
   - `loginUrl`: ennek a `login.php`-ra kell vezetnie.
   - `activityUrl`: ennek a `hitnrun.php`-ra kell vezetnie.
   
2. **Login adatok kitöltése:** A `loginData` változókat töltsd ki a felhasználóneveddel és a jelszavaddal.

3. **Kimeneti könyvtár beállítása:** 
   - Az `outputDir` változó határozza meg, hova kerülnek a `.torrent` fájlok.
   - **Figyelem:** Ez az alkalmazás **nem tölti vissza a torrentet**, hanem egy tetszőleges torrent alkalmazásba kell beállítani, hogy ugyanabba a mappába töltse le a torrent fájlokat, ahol automatikusan hozzáadja őket a torrent alkalmazáshoz.

4. **Alkalmazás buildelése:**
   - Windows rendszeren: `go build -o ncore_automation.exe main.go`
   - Linux rendszeren: `go build -o ncore_automation main.go`
   - Futtasd le a fenti parancsot a letöltött fájl mappájában.

5. **Alkalmazás futtatása:**
   - A program ezután már futtatható vagy ütemezhető, ez már a felhasználó választása. 😄

---

**Fontos:** Semmilyen felelősséget nem vállalok jogsértő tartalom terjesztéséért, ez az alkalmazás csak a manuális lépéseket váltja ki.
