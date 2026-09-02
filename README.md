# 🧱 GoCraft 3D — 1:1 Minecraft Survival & Voxel Engine in Go

Ein vollständiger, hardware-beschleunigter **1:1 Minecraft Survival & Voxel Klon** in Go mit `raylib-go` (500+ FPS).

---

## 🚀 Spiel starten

Führe die `.exe` aus oder starte per PowerShell:
```powershell
.\start.ps1
```
*(oder `.\gocraft.exe` / `.\minecraft.exe`)*

---

## 🎨 Eigenes Texture Pack einbinden (Sehr gerne!)

**Ja, ein echtes Texture Pack hilft enorm!**
Das Spiel unterstützt jetzt automatisches Laden von Texture Packs:
- Lege einfach deine Texturdatei (z. B. `terrain.png`, `atlas.png` oder `texturepack.png`) in den Ordner `assets/` oder direkt in den Hauptordner `D:\Downloads\Go\`.
- Das Spiel erkennt das Texture Pack automatisch beim Start und lädt alle HD- oder Original-Minecraft-Texturen!

---

## 🎮 Steuerung & Tastenbelegung

| Taste / Maus | Aktion |
|---|---|
| **W, A, S, D** | Laufen / Bewegen |
| **LEERTASTE (Space)** | Springen / Nach oben schwimmen |
| **STRG / SHIFT** | Sprinten (mit dynamischem FOV-Geschwindigkeitseffekt) |
| **C** / **SHIFT unter Wasser** | Schleichen / Nach unten abtauchen |
| **MAUSBEWEGUNG** | 360° First-Person Umschauen |
| **LINKE MAUSTASTE (Gedrückt halten)** | **Block abbauen** (mit echten Riss-Stufen & Sound wie in Minecraft) |
| **RECHTE MAUSTASTE** | **Block platzieren** / **Werkbank (3x3 Crafting) öffnen** |
| **E** | **Spieler-Inventar (2x2 Crafting) öffnen / Werkbank schließen** |
| **G** | **Spielmodus wechseln** (Survival-Modus $\leftrightarrow$ Kreativ-Modus) |
| **1 – 9** / **Mausrad** | Hotbar-Slot auswählen |
| **F5** | **Kamera-Perspektive umschalten** (1st-Person $\leftrightarrow$ 3rd-Person Rücken $\leftrightarrow$ 3rd-Person Front) |

---

## 🛠️ Survival & Inventar-Mechaniken

1. **Echter Survival-Start**:
   - Man startet **vollständig ohne Items** (leeres Inventar) und baut sich Schritt für Schritt durch Holzfällen, Crafting und Werkzeuge alles selbst auf.

2. **Echtes Abbau-System (Mining Cracks)**:
   - Blöcke zerbrechen nicht sofort, sondern benötigen je nach Härte (Erde: ~0.5s, Holz: ~1.5s, Stein: ~2.8s) kontinuierliches Halten der linken Maustaste mit sichtbaren Rissen und Schlag-Sounds.
   - Im Kreativmodus (`G`) werden Blöcke sofort mit 1 Klick abgebaut.

3. **Präzises Stack- & Inventar-Handling**:
   - **Linksklick auf Slot**: Nimmt den ganzen Stack auf / legt ihn ab.
   - **Rechtsklick auf Stack**: Nimmt genau die **Hälfte des Stacks** auf.
   - **Rechtsklick mit gehaltenem Item**: Platziert genau **1 einzelnes Item** in den Ziel-Slot.


2. **2x2 & 3x3 Crafting-System**:
   - **2x2 Inventar-Crafting (`E`)**:
     - 1 Holzstamm $\rightarrow$ 4 Holzbretter
     - 4 Holzbretter $\rightarrow$ 1 Werkbank (Crafting Table)
     - 1 Kohle + 1 Holzbrett $\rightarrow$ 4 Fackeln
     - 4 Sand $\rightarrow$ 4 Sandstein
     - 1 Bruchstein + 1 Eichenblätter $\rightarrow$ 1 Moosiger Bruchstein
     - 4 Erde + 4 Stein $\rightarrow$ 4 Ziegelsteine
   - **3x3 Werkbank-Crafting (Rechtsklick auf Werkbank)**:
     - 8 Bruchstein (Ring-Form) $\rightarrow$ 1 Ofen
     - 6 Holzbretter + Erze $\rightarrow$ 1 Bücherregal
     - 4 Redstone + 5 Sand $\rightarrow$ 1 TNT
     - Alle 2x2 Rezepte funktionieren auch in der 3x3 Werkbank!

3. **Texturierte First-Person-Hand & Items**:
   - In der First-Person-Sicht wird das gehaltene Item als **echter 3D-Voxel-Miniaturblock** mit allen authentischen 16x16-Texturen aus dem Textur-Atlas gerendert.
   - Flüssige Schwung- und Schlaganimation bei Klickaktionen.

4. **Hardware-GPU-Beschleunigung**:
   - Vollständiges VBO/VAO-Caching direkt im VRAM deiner GPU für stabile **500+ FPS**.
   - Hardware-GPU Backface Culling und Multi-Pass Alpha-Transparenz.


---

## 🌍 Features & Grafik-Optimierungen:

1. **High-Definition 16x16 Pixel Art Textur-Atlas**:
   - Authentische Minecraft-Texturen für alle Blöcke (Gras, Rinde, Jahresringe, Holzmaserung, funkelnde Erze, Ziegel, Ofen, Fackeln, Bücherregale, TNT, Werkbank).
   - **Sub-Pixel UV Inset**: Vollständige Beseitigung von Nahtstellen und Kantenlinien zwischen Blöcken.

2. **Erweitertes Block- & Erz-System (26 Blocktypen)**:
   - Grasblock, Erde, Stein, Bruchstein (Cobblestone), Moosiger Bruchstein, Grundgestein, Eichenholzstamm, Eichenholzbretter, Blätter (Cutout), Glas (transparent), Sand, Sandstein.
   - **Alle Erzadern**: Kohle-, Eisen-, Gold-, **Diamant-**, **Redstone-**, **Smaragd-** und **Lapislazuli-Erz**!
   - Ziegelsteine, Werkbank, **Ofen**, **Bücherregal**, **3D-Fackeln** und **TNT**!

3. **Multi-Pass Transparenz- & Wasser-Rendering**:
   - Echte Trennung in **Opaque-**, **Alpha-Cutout-** (Blätter/Glas/Fackeln) und **Translucent Water-**Pässe.
   - **Water-to-Water Culling**: Interne Wasserpolygone werden gefiltert für glasklares Wasser ohne Verdeckungsfehler.

4. **Atmosphäre & Effekte**:
   - **3D-Voxel-Wolkenschicht** auf y=42, die dynamisch über die Welt zieht.
   - **Block-Abbau-Partikel**: Physikalische 3D-Splitter beim Zerschlagen von Blöcken.
   - Quadratische Sonne und Mond mit weichem Tag-, Sonnenuntergangs- und Nacht-Farbverlauf.

5. **Performance & Frustum-Culling**:
   - Chunks hinter der Kamera werden automatisch gefiltert.
   - Zero-Allocation Chunk-Meshing für flüssige 60-144+ FPS.

6. **Steve-Charakter & First-Person-Hand**:
   - Detailliertes Steve-Modell mit pixel-genauem Gesicht, Shirt, Jeans und Schuhen.
   - 3D-Mini-Voxel-Modell in der Hand für das aktuell ausgewählte Item.

