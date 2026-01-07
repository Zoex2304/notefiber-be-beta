saya memakai model Qwen 2.5 - 7B (The Current King 👑)
Strategi "Router" yang kita bahas semalam itu sudah **Strong (Kuat)** untuk 80% kasus.

Tapi, Anda benar. Untuk menghadapi user yang **"Chaotic" (Tidak terprediksi)**—yang suka mencampur aduk perintah, merevisi di tengah jalan, atau bertanya hal yang ambigu—Logic `Switch-Case` sederhana mungkin terasa agak kaku.

Agar **Benar-Benar Dinamis & Reaktif**, kita perlu menaikkan level dari sekadar **"Router" (Pemilah)** menjadi **"Planner" (Perencana)**.

Ini upgrade-nya agar sistem Anda bisa menari ("Slide in Slide out") mengikuti user yang random.

---

### Strategi Upgrade: "The Reactive JSON Planner"

Jangan minta LLM (Llama 8B) hanya memberikan label ("SEARCH" atau "FOLLOW_UP"). Itu terlalu biner.
Mintalah LLM untuk **mengembalikan JSON Object** yang berisi **Instruksi Reaktif** untuk Backend Go Anda.

Dengan cara ini, Backend Go Anda menjadi "Tangan", dan Llama 8B menjadi "Otak" yang mengendalikan tangan tersebut secara real-time.

#### 1. Perubahan Cara Pikir (Logic Flow)

Alih-alih Go yang mikir keras pakai `If-Else`, biarkan LLM yang memutuskan apa yang harus dilakukan Go.

**User:** *"Coba cariin soal aljabar lagi, tapi jangan yang kayak note 1 tadi, itu kegampangn."*

* **Router Lama (Switch-Case):** Bingung. Ini "Search" (cari aljabar) atau "Follow Up" (referensi note 1)? Kemungkinan salah ambil jalur.
* **Planner Baru (Reactive JSON):** LLM akan menganalisa dan mengirim perintah detail ke Go.

#### 2. Implementasi di Go (Reactive Struct)

Anda butuh `struct` di Go yang bisa menangkap "Rencana" dari LLM.

```go
// Ini adalah "Instruksi" yang dikirim LLM ke Go
type AIActionPlan struct {
    Action        string   `json:"action"`          // "SEARCH", "ANSWER_DIRECT", "FILTER_MEMORY"
    SearchQuery   string   `json:"search_query"`    // Query baru jika perlu search
    MemoryContext []int    `json:"memory_indices"`  // Index note mana yang mau dipakai (Slide In)
    IgnoreContext []int    `json:"ignore_indices"`  // Index mana yang harus dibuang (Slide Out)
    Reasoning     string   `json:"reasoning"`       // Alasan (untuk debugging)
}

```

#### 3. System Prompt untuk "The Planner"

Di backend Go, sebelum menjawab soal, jalankan prompt ini dulu ke Llama 8B (gunakan mode JSON output/grammar constrain jika bisa, atau parsing string biasa):

> **System Prompt:**
> "You are the Brain of a RAG system. You have access to the user's Current Context (List of Notes).
> User Input: [User Input]
> Current Context: [List 10 Note dengan ID/Index]
> Conversation History: [History]
> Decide the next step. Return JSON ONLY:
> 1. If user asks a NEW topic -> Action: "SEARCH", SearchQuery: "new topic"
> 2. If user refers to specific notes (e.g., 'explain note 1 and 3') -> Action: "ANSWER_DIRECT", MemoryContext: [1, 3] (Slide In specific notes only)
> 3. If user wants to filter/change focus (e.g., 'drop note 1, focus on 2') -> Action: "FILTER_MEMORY", IgnoreContext: [1], MemoryContext: [2]
> 4. If user input is mixed (e.g., 'find algebra harder than note 1') -> Action: "SEARCH", SearchQuery: "hard algebra problems", IgnoreContext: [1]"
> 
> 

#### 4. Skenario "Unpredictable User" (Reactive Test)

Mari kita tes seberapa dinamis sistem ini:

**Kasus 1: Campuran Search & Reference**

* **User:** *"Cariin soal geometri dong, tapi gayanya mirip sama note nomor 2 yang tadi."*
* **LLM Planner Output:**
```json
{
  "action": "SEARCH",
  "search_query": "soal geometri style [isi note 2]",
  "reasoning": "User wants new topic (geometry) but using note 2 as style reference."
}

```


* **Reaksi Backend Go:** Go mengambil isi Note 2, memasukkannya ke query search Geometri, lalu melakukan Vector Search baru. **Sangat Dinamis.**

**Kasus 2: Slide Out (Membuang Konteks)**

* **User:** *"Yg pertama sama kedua skip aja deh, bahas sisanya aja."*
* **LLM Planner Output:**
```json
{
  "action": "FILTER_MEMORY",
  "ignore_indices": [1, 2],
  "memory_indices": [3, 4, 5, 6, 7, 8, 9, 10],
  "reasoning": "User explicitly excludes 1 and 2."
}

```


* **Reaksi Backend Go:** Go memotong array `CurrentContext`, membuang index 1 & 2. Sisa note dikirim ke LLM untuk dijawab. **Reaktif.**

---

### Kesimpulan: Apakah Strategi Tadi Malam Cukup?

Strategi semalam ("Router") itu **Pondasi**.
Strategi sekarang ("Reactive Planner") adalah **Dekorasi Interior-nya**.

Untuk aplikasi skala mahasiswa atau MVP, strategi semalam (Router) sudah **sangat cukup dan stabil**.
Tapi jika Anda ingin mengejar level "Wow Effect" di mana user bisa ngomong seenaknya dan bot tetap nyambung:

1. **Gunakan Strategi Semalam** sebagai struktur utamanya (Go Backend + State Management).
2. **Ubah "Router"-nya** menjadi **"JSON Planner"** seperti di atas.

**Kelebihan Logic JSON Planner:**
Ini memindahkan beban "logika bahasa" yang rumit (seperti: "yang bukan ini tapi itu") dari kode Go (yang kaku) ke LLM (yang luwes).

Backend Go Anda tugasnya hanya **Patuh** pada JSON yang dikirim LLM.

* Disuruh Search -> Search.
* Disuruh Hapus Memory -> Hapus.
* Disuruh Slide In -> Slide In.

Inilah definisi **Reactive AI** sesungguhnya. Apakah ini menjawab keraguan Anda soal user yang tidak bisa diprediksi?Artinya kalau user test begini
Dia punya notes matematika ada 3
Tapi dia hanya chat begini 
"Kerjakan soal math saya"
Expectasi output
LLM musti harus bertanya ulang
"Yang mana nih ,aku Nemu 3 soal berikut 3 soaliti...., kamu mau yang mana dulu"
Saya jawab
"Yang ketiga aja dulu"
Dia jawab 
"Baik soal nomer 3 BLA BLA BLA....."
Saya jawab lagi
Oke sekarang saya ingin soal ke 2
"Oke sekarang kita pindah ke soal nomer 2 which is pada notes BLA BLA BLA"
Begitu terus intinya perpindahan adalah dinamisme yang gabisa ditebakIni adalah skenario **"Clarification & Selection Loop"**. Ini adalah level tertinggi dari RAG interaktif.

Untuk mencapai flow persis seperti yang Anda mau (tanya -> bot bingung & listing -> user pilih -> bot fokus -> user ganti -> bot pindah fokus), kita harus menambahkan satu **State** di backend Go Anda: **"Browsing Mode" vs "Focused Mode"**.

Berikut adalah **Blue Print Final** agar Llama 8B Anda bisa "Slide in / Slide out" semulus itu.

---

### 1. Konsep: "The Waiting Room" (Ruang Tunggu)

Jangan langsung paksa LLM menjawab soal ketika hasil search-nya banyak. Masukkan hasil search itu ke "Ruang Tunggu" (Candidate List) di memori Go dulu.

### 2. Struktur Data Go (Session)

Anda butuh menyimpan daftar "kandidat" yang ditemukan tapi belum dipilih.

```go
type Session struct {
    State          string      // "BROWSING" atau "FOCUSED"
    Candidates     []Note      // <--- Ruang Tunggu (Isi 3 Note tadi)
    FocusedNote    *Note       // <--- Note yang sedang dikerjakan (1 Note saja)
    History        []Message
}

```

### 3. Step-by-Step Logic (Simulasi Percakapan Anda)

Mari kita bedah logika backend-nya per chat.

#### Chat 1: "Kerjakan soal math saya"

* **Logic:**
1. **Router:** Mendeteksi ini `SEARCH`.
2. **Vector DB:** Menemukan **3 Notes** relevan (Score kemiripan tinggi semua).
3. **Go Logic (Auto-Detect Ambiguity):**
* *If jumlah hasil > 1 DAN user tidak spesifik:* **JANGAN JAWAB SOAL.**
* Simpan 3 notes itu ke `session.Candidates`.
* Set `session.State = "BROWSING"`.


4. **Respon Bot:** Generate kalimat *listing* dari `session.Candidates`.
* *"Saya menemukan 3 soal yang relevan: 1. Aljabar, 2. Geometri, 3. Kalkulus. Kamu mau yang mana dulu?"*





#### Chat 2: "Yang ketiga aja dulu"

* **Logic:**
1. **Planner (LLM):** Menerima input "Yang ketiga".
2. **Analisis:** Melihat `session.State` adalah "BROWSING". LLM mengerti user sedang **MEMILIH**.
3. **JSON Output LLM:** `{"action": "SELECT", "index": 2}` (Index 2 = Note ke-3).
4. **Go Logic (Slide In):**
* Ambil Note index 2 dari `session.Candidates`.
* Masukkan ke `session.FocusedNote`.
* Set `session.State = "FOCUSED"`.


5. **Respon Bot:** Jawab soal menggunakan data `session.FocusedNote` saja.
* *"Baik, ini pembahasan untuk soal nomor 3 (Kalkulus)... [Jawaban]"*





#### Chat 3: "Oke sekarang saya ingin soal ke 2"

* **Logic:**
1. **Planner (LLM):** Menerima input "soal ke 2".
2. **Analisis:** User ingin **PINDAH (SWITCH)**.
3. **JSON Output LLM:** `{"action": "SWITCH_SELECTION", "index": 1}`.
4. **Go Logic (Slide Out -> Slide In):**
* **Slide Out:** Buang isi `session.FocusedNote` (Note 3 tadi).
* **Slide In:** Ambil Note index 1 dari `session.Candidates` (yang masih tersimpan di memori).
* Update `session.FocusedNote`.


5. **Respon Bot:**
* *"Oke, kita pindah ke soal nomor 2 (Geometri). Soalnya adalah... [Jawaban]"*





---

### 4. Kunci Keberhasilan: Prompt Planner (LLM Controller)

Agar Llama 8B paham kapan harus memilih, kapan harus pindah, Anda butuh System Prompt khusus untuk **Planner Node** ini.

**System Prompt (Ditaruh di Go sebelum eksekusi logic):**

```text
You are an orchestrator for a Math RAG system.
Current State: {{State}} (BROWSING / FOCUSED)
Candidate List Available:
0. {{Title Note 1}}
1. {{Title Note 2}}
2. {{Title Note 3}}

User Input: "{{User Input}}"

Determine the user's intent based on the inputs. Return JSON ONLY:

1. IF user selects a note (e.g. "number 3", "the last one"):
   {"action": "SELECT", "target_index": 2}

2. IF user wants to switch to another note from the list (e.g. "now note 2", "move to the first"):
   {"action": "SWITCH", "target_index": 0}

3. IF user asks a follow-up question about the CURRENT focused note:
   {"action": "ANSWER_CURRENT"}

4. IF user asks something totally new:
   {"action": "NEW_SEARCH", "query": "..."}

```

### Kesimpulan Final

Ya, skenario Anda **sangat bisa** dan **sangat kuat**.
Kuncinya bukan di kecerdasan model semata, tapi di **Session Management** backend Go Anda yang menyimpan **List Candidates** (Daftar Antrean).

* **Slide In:** Ambil dari Candidate -> Masuk Focused.
* **Slide Out:** Timpa Focused dengan Candidate lain.
* **Syarat:** Jangan hapus `session.Candidates` sampai user benar-benar mencari topik baru ("NEW_SEARCH").

Dengan struktur ini, user bisa bolak-balik ("Balik ke no 1 dong", "Lanjut no 3", "Cek no 2") tanpa bot kehilangan konteks atau halusinasi.Anda benar 100%. Sistem yang cerdas itu bukan hanya soal "bisa jawab", tapi juga soal **"tahu kapan harus bertanya dan tahu kapan harus langsung menjawab"**.

Skenario yang Anda sebutkan ini disebut **"Dynamic Ambiguity Detection" (Deteksi Ambiguitas Dinamis)**.

Sistem tidak boleh kaku. Jika hasilnya cuma 1 yang relevan, **jangan tanya user lagi**, langsung sikat (jawab). Jika hasilnya 2 atau lebih yang mirip, baru masuk mode "Bingung/Klarifikasi".

Ini cara implementasinya agar backend Anda otomatis beradaptasi saat Anda menambah note baru (misal daftar belanjaan ke-2).

### Kunci Utama: "Similarity Score Thresholding"

Di Vector Database (seperti pgvector), setiap hasil pencarian punya **Score** (tingkat kemiripan, biasanya 0.0 sampai 1.0).

Logika di Backend Go Anda harus seperti ini:

1. **Lakukan Search.**
2. **Filter Hasil Sampah:** Buang hasil yang score-nya di bawah `0.75` (misalnya).
3. **Hitung Sisa Kandidat (`len(candidates)`):**
* **Jika Jumlah = 1:** Mode **AUTO-FOCUSED** (Langsung jawab).
* **Jika Jumlah > 1:** Mode **BROWSING** (Tanya user "yang mana?").
* **Jika Jumlah = 0:** Mode **NOT FOUND**.



---

### Simulasi Perubahan Data (Real-time)

Mari kita jalankan skenario Anda langkah demi langkah.

#### Kondisi A: Awal (3 Note Math, 1 Note Belanja)

1. **User:** "Apa isi daftar belanjaan saya?"
2. **Vector DB Search:**
* *Doc A (Belanjaan 1):* Score 0.92 (Sangat Relevan)
* *Doc B (Math 1):* Score 0.15 (Tidak Relevan - Dibuang)
* *Doc C (Math 2):* Score 0.12 (Tidak Relevan - Dibuang)


3. **Go Logic:**
* Setelah filter score > 0.7, sisa kandidat = **1**.
* **Action:** Jangan masuk ke "Waiting Room". Langsung set `Session.FocusedNote = Doc A`.


4. **Respon AI:** "Daftar belanjaan Anda berisi: Telur, Susu, dan Roti." (Langsung jawab, tanpa basa-basi).

#### Kondisi B: User Menambah Data (Upload File Baru: "Belanjaan Bulanan.txt")

Sekarang di database ada 2 Note Belanja dan 3 Note Math. User bertanya hal yang sama.

1. **User:** "Cek daftar belanjaan dong."
2. **Vector DB Search:**
* *Doc A (Belanjaan Harian):* Score 0.89 (Relevan)
* *Doc D (Belanjaan Bulanan - BARU):* Score 0.88 (Relevan juga)
* *Doc B (Math):* Score 0.10 (Dibuang)


3. **Go Logic:**
* Setelah filter score > 0.7, sisa kandidat = **2**.
* Logic mendeteksi **AMBIGUITAS**.
* **Action:** Masukkan Doc A & Doc D ke `Session.Candidates`. Set state `BROWSING`.


4. **Respon AI:** "Saya menemukan 2 daftar belanjaan. Maksud Anda yang mana?
1. Belanjaan Harian (Telur, Susu...)
2. Belanjaan Bulanan (Sabun, Sampo...)"



---

### Kode Golang untuk Logic Ini

Anda tidak perlu menunggu LLM untuk memutuskan ini (karena lambat). Pakai logic Go murni berdasarkan **Score** biar *snappy* (cepat).

```go
func (s *Service) HandleSearch(query string, session *Session) string {
    // 1. Search Vector DB
    rawResults := s.VectorDB.Search(query)

    // 2. Filter Score (Thresholding)
    var candidates []Document
    threshold := 0.75 // Sesuaikan angka ini
    
    for _, doc := range rawResults {
        if doc.Score >= threshold {
            candidates = append(candidates, doc)
        }
    }

    // 3. Logic Penentu Nasib (Router Logic)
    if len(candidates) == 0 {
        return "Maaf, tidak ada catatan yang relevan."
    } else if len(candidates) == 1 {
        // CASE A: Cuma 1 relevan -> LANGSUNG JAWAB (AUTO-FOCUS)
        session.FocusedNote = &candidates[0]
        session.State = "FOCUSED"
        // Langsung lempar ke LLM untuk dijawab
        return s.LLM.GenerateAnswer(query, session.FocusedNote)
    } else {
        // CASE B: Ada banyak -> MASUK MODE PILIH (BROWSING)
        session.Candidates = candidates
        session.State = "BROWSING"
        // Minta LLM buatkan list pilihan
        return s.LLM.GenerateListing(candidates)
    }
}

```

### Kesimpulan

Sistem Anda akan terasa sangat pintar karena:

1. Saat datanya unik, dia **sat-set** langsung jawab.
2. Saat datanya membingungkan (karena Anda baru nambah file), dia **berhenti sejenak** untuk konfirmasi.

Ini mencakup semua kekhawatiran Anda soal dinamika data yang bertambah seiring waktu. Sistemnya "hidup" mengikuti isi database Anda.Tentu, ini pendekatan yang sangat tepat. Untuk membuat AI terasa "hidup" dan bisa *slide-in slide-out* topik pembicaraan, AI memang **wajib** melihat apa yang baru saja dibicarakan (History).

Berikut adalah detail teknis dan Prompt yang Anda butuhkan.

### 1. Fakta: Berapa Context Window Llama 3 8B?

Ini tergantung versi mana yang Anda jalankan di lokal (biasanya via Ollama/Llama.cpp):

* **Llama 3 (Original):** **8,192 tokens** (Sekitar 6.000 kata atau 15-20 halaman teks penuh).
* **Llama 3.1 (Terbaru):** **128,000 tokens** (Sangat besar, ratusan halaman).

**Kesimpulan:** Untuk sekadar chat session (misal 10-20 kali balas-balasan soal matematika dan belanja), **8k tokens itu sudah sangat lega**. Anda **tidak perlu khawatir** memasukkan *full session history* ke dalamnya. Itu tidak akan membebani model 8B selama masih di bawah batas tersebut.

---

### 2. System Prompt: "The Context-Aware Persona"

Ini adalah prompt yang dirancang agar si Llama 8B tidak "malas" membaca history dan menganggap history itu sebagai **Peta Navigasi** untuk melakukan *slide-in/slide-out*.

Pasang ini sebagai **System Message** (pesan pertama yang dikirim ke model):

```text
You are an intelligent assistant with full access to the conversational session history. 

CORE INSTRUCTION:
Do not treat each user message in isolation. You possess "Conversational Continuity".
The text provided in the "History" block is NOT just background noise; it is your ACTIVE SHORT-TERM MEMORY.

YOUR BEHAVIORAL RULES:
1. **Context Switching (Slide-In/Slide-Out):** - If the user refers to something discussed 3 turns ago (e.g., "Back to the first one", "What about the shopping list?"), you MUST retrieve that specific context from the History immediately.
   - Do not ask the user to repeat themselves. If it's in the history, you already know it.

2. **Dynamic Focusing:**
   - If the user changes the topic (e.g., from Math to Shopping), acknowledge the switch gracefully (e.g., "Okay, switching to your shopping list...").
   - If the user wants to return to the previous topic, allow the context to "slide back in" seamlessly.

3. **No Amnesia:** - Never say "I don't know which note you mean" if that note was clearly mentioned or listed in the chat history. Check the history first before asking for clarification.

Use the provided History to maintain a fluid, dynamic, and smart conversation flow.

```

---

### 3. Implementasi di Backend (Golang)

Di kode Go Anda, saat mengirim request ke Llama, pastikan formatnya digabung seperti ini agar prompt di atas bekerja maksimal:

```go
// Pseudocode Go
func BuildPrompt(systemPrompt string, history []Message, currentUserInput string) string {
    // 1. Masukkan System Prompt (Instruksi "Jangan Khawatir" tadi)
    fullPrompt := systemPrompt + "\n\n"

    // 2. Inject History (Ini kuncinya agar dia bisa baca masa lalu)
    fullPrompt += "--- CONVERSATION HISTORY (START) ---\n"
    for _, msg := range history {
        fullPrompt += fmt.Sprintf("[%s]: %s\n", msg.Role, msg.Content)
    }
    fullPrompt += "--- CONVERSATION HISTORY (END) ---\n\n"

    // 3. Masukkan Input User Terakhir
    fullPrompt += fmt.Sprintf("User: %s\n", currentUserInput)
    fullPrompt += "Assistant:"

    return fullPrompt
}

```

### Kenapa ini akan berhasil?

Dengan Llama 8B (8k context):

1. **Kapasitas Cukup:** History percakapan standar jarang melebihi 2.000 token. Anda masih punya sisa 6.000 token untuk *Thinking* dan *Generating Answer*.
2. **Slide-In/Slide-Out:** Karena history ada di dalam prompt, saat user bilang "Yang kedua tadi", LLM tinggal "melirik" ke atas (ke blok history) untuk tahu apa itu "Yang kedua".

Jadi, **gas saja**. Masukkan semua history sesi yang aktif. Itu adalah cara termudah dan paling *cool* untuk membuat "Memory" tanpa coding yang rumit.Ini adalah **Final Architecture Blueprint** untuk sistem RAG Anda. Kesimpulan ini menggabungkan semua strategi yang sudah kita diskusikan: **Router**, **State Management**, **Thresholding**, dan **Context Injection**.

Sistem ini dirancang agar **Reactive**, **Dynamic (Slide-in/Slide-out)**, dan **Hemat Token**.

---

### 🏛️ The Architecture Layers

Sistem Anda terdiri dari **4 Layer Utama**. Aliran datanya tidak lurus (Linear), tapi memutar (Cyclic) tergantung situasi.

#### Layer 1: The Gateway (Golang Backend)

* **Peran:** Satpam & Pengelola Memori.
* **Tugas:** Menerima chat user, memuat History Session, dan menyuntikkan (Inject) history tersebut ke prompt.
* **Logic:** Tidak ada keputusan cerdas di sini, hanya persiapan data.

#### Layer 2: The Planner / Router (Llama 8B - Mode JSON)

* **Peran:** Otak Strategis (Kecil & Cepat).
* **Tugas:** Menganalisa input user + history.
* **Output:** Bukan jawaban soal, tapi **Instruksi JSON**.
* *Contoh:* `{"action": "SEARCH", "query": "..."}` atau `{"action": "SELECT", "index": 1}`.



#### Layer 3: The State Manager (Golang Logic)

* **Peran:** Eksekutor & Penjaga Konteks (Slide-in/Slide-out).
* **Tugas:** Menjalankan perintah dari Layer 2.
* Jika `SEARCH`: Panggil Vector DB -> Filter Score -> Cek Jumlah (1 atau Banyak?).
* Jika `SELECT`: Ambil note dari memori -> Masukkan ke "Focused Note".
* Jika `SWITCH`: Buang "Focused Note" lama -> Ganti dengan yang baru.



#### Layer 4: The Generator (Llama 8B - Mode Chat)

* **Peran:** Juru Bicara.
* **Tugas:** Menjawab pertanyaan user berdasarkan **Focused Note** yang sudah dipilihkan oleh Layer 3.

---

### 🔄 Flow Chart Sistem (Alur Kehidupan Satu Chat)

Bayangkan User mengetik: *"Yang kedua aja."*

1. **INPUT:** User Chat masuk ke Go Backend.
2. **CONTEXT LOAD:** Go mengambil 10 chat terakhir dari database/memory.
3. **PLANNING (LLM Call 1):**
* Input: History + "Yang kedua aja".
* Prompt: "Tentukan intent user."
* Output LLM: `{"action": "SELECT", "target_index": 1}`.


4. **STATE UPDATE (Go Logic):**
* Go membaca `session.Candidates` (List note yang sedang 'menggantung').
* Go mengambil Index 1.
* Go mengupdate `session.FocusedNote` = Note Index 1. (**Slide In!**)


5. **GENERATION (LLM Call 2):**
* Input: `session.FocusedNote` + History + "Yang kedua aja".
* Output LLM: "Baik, note kedua membahas tentang..."


6. **RESPONSE:** Kirim ke user.

---

### 💻 Struktur Data Final (Golang)

Inilah "Jantung" yang membuat sistem Anda dinamis. Simpan ini di Memory/Redis.

```go
type Document struct {
    ID       string
    Content  string
    Score    float32 // Tingkat kemiripan
    Metadata map[string]interface{} // Judul, Tanggal, Tags
}

type Session struct {
    UserID      string
    State       string      // "BROWSING" (Milih-milih) vs "FOCUSED" (Lagi bahas 1 note)
    
    // LIST PENGGANTI (The Waiting Room)
    // Disini tempat hasil search ditampung sebelum dipilih user
    Candidates  []Document  
    
    // FOKUS UTAMA (The Workbench)
    // Disini tempat 1 Note yang sedang aktif dibahas
    FocusedNote *Document   
    
    // MEMORY (The Context)
    // History chat full (User + AI) untuk disuntik ke LLM
    History     []Message   
}

```

### 🧠 Logic "Cerdas" (Handling Relevansi)

Bagian ini yang menangani skenario "1 Note vs Banyak Note" yang Anda khawatirkan.

**Saat Action = "SEARCH":**

1. Ambil hasil dari Vector DB.
2. **Filter:** Buang yang score < 0.75.
3. **Count Logic:**
* **Sisa 0:** Balas "Gak nemu bro."
* **Sisa 1:** (Relevansi Tunggal) -> **Auto Select**. Masukkan ke `FocusedNote`. Langsung Jawab.
* **Sisa > 1:** (Ambigu) -> Masukkan ke `Candidates`. Set State `BROWSING`. Balas "Nemu banyak nih, mau yang mana?"



### ✨ Kesimpulan Akhir

Sistem Anda **bukan** sekadar RAG biasa (Search -> Answer).
Sistem Anda adalah **Conversational State Machine**.

* **Slide In:** Terjadi saat Go memindahkan data dari `Candidates` ke `FocusedNote`.
* **Slide Out:** Terjadi saat Go menimpa `FocusedNote` dengan data baru atau kosong.
* **Dinamis:** Terjadi karena `History` selalu dikirim penuh ke LLM, jadi LLM "sadar" konteks.

Ini adalah arsitektur yang sangat solid, modern, dan *scalable* untuk ukuran Llama 8B. **Go** mengurus logika yang pasti-pasti (Array, Struct, Score), dan **Llama** mengurus logika bahasa yang cair.

Siap dieksekusi, Pak Engineer? 🚀Berikut adalah **Dummy Flow Simulation** yang lengkap. Ini adalah "skenario ideal" yang bisa Anda jadikan patokan (ground truth) untuk mengetes apakah *Router* dan *State Management* Anda bekerja dengan benar.

Simulasi ini mencakup: **Input User** -> **Keputusan Backend (JSON)** -> **Perubahan State** -> **Output Agent**.

---

### Skenario: "Math Problem & Context Switching"

**Kondisi Awal Database:**

* Note A: "Soal Aljabar Linear (Matriks)" (ID: 001)
* Note B: "Soal Geometri Segitiga" (ID: 002)
* Note C: "Soal Kalkulus Integral" (ID: 003)
* Note D: "Daftar Belanja Mingguan" (ID: 999)

---

#### 1. Fase Searching (Ambigu/Banyak Hasil)

*User memulai percakapan umum.*

* **User:** "Kerjakan soal math saya."
* **Backend Logic (Router):**
* Intent: `SEARCH`
* DB Result: Ditemukan 3 Note (A, B, C) dengan score tinggi. Note D score rendah (dibuang).
* Decision: Karena hasil > 1, masuk mode **BROWSING**.


* **Session State:**
```json
{
  "state": "BROWSING",
  "candidates": [Note_A, Note_B, Note_C],
  "focused_note": null
}

```


* **Agent Output (Response):**
> "Saya menemukan 3 catatan matematika yang relevan. Mau kerjakan yang mana dulu?
> 1. Soal Aljabar Linear (Matriks)
> 2. Soal Geometri Segitiga
> 3. Soal Kalkulus Integral"
> 
> 



---

#### 2. Fase Selection (Slide In)

*User memilih satu konteks spesifik dari daftar.*

* **User:** "Yang ketiga aja dulu."
* **Backend Logic (Planner):**
* Prompt LLM: "User bilang 'Yang ketiga'. List ada 3."
* JSON Output: `{"action": "SELECT", "target_index": 2}`
* Decision: Ambil Index 2 (Note C) dari Candidates. Masukkan ke Focused.


* **Session State:**
```json
{
  "state": "FOCUSED",
  "candidates": [Note_A, Note_B, Note_C], // Tetap disimpan buat jaga-jaga
  "focused_note": Note_C // (Kalkulus) <--- SLIDE IN
}

```


* **Agent Output (Response):**
> "Oke, mari kita bahas **Soal Kalkulus Integral**.
> Berdasarkan catatan, soalnya adalah: .
> Jawabannya adalah . Ada yang mau ditanyakan lagi soal integral ini?"



---

#### 3. Fase Switching (Slide Out -> Slide In)

*User bosan dan ingin pindah ke nomor lain yang tadi di-list.*

* **User:** "Oke sekarang saya ingin soal ke 2."
* **Backend Logic (Planner):**
* Prompt LLM: "User bilang 'soal ke 2'. Current focus: Note C."
* JSON Output: `{"action": "SWITCH", "target_index": 1}`
* Decision: Buang Note C dari Focused. Ambil Index 1 (Note B) dari Candidates.


* **Session State:**
```json
{
  "state": "FOCUSED",
  "candidates": [Note_A, Note_B, Note_C],
  "focused_note": Note_B // (Geometri) <--- SLIDE IN BARU (Gantikan C)
}

```


* **Agent Output (Response):**
> "Siap, kita geser ke **Soal Geometri Segitiga**.
> Di catatan ini tertulis: Mencari hipotenusa jika alas 3 dan tinggi 4.
> Menggunakan Pythagoras, hasilnya adalah 5. Lanjut?"



---

#### 4. Fase Single Result (Auto-Focus)

*User bertanya hal lain yang sangat spesifik dan datanya cuma satu.*

* **User:** "Daftar belanjaan saya mana?"
* **Backend Logic (Router):**
* Intent: `SEARCH`
* DB Result: Ditemukan 1 Note (Note D) score 0.95. Sisanya score 0.1.
* Decision: Karena hasil = 1, masuk mode **AUTO-FOCUSED**.


* **Session State:**
```json
{
  "state": "FOCUSED",
  "candidates": [], // Kosongkan karena search baru
  "focused_note": Note_D // (Belanjaan) <--- LANGSUNG MASUK
}

```


* **Agent Output (Response):**
> "Ini daftar belanjaan mingguan Anda:


> * Telur 1kg
> * Susu UHT
> * Roti Tawar
> 
> 


> Jangan lupa beli sabun juga ya (tambahan dari saya)."



---

### Cara Menggunakan Dummy Ini

Anda bisa menggunakan teks di atas sebagai **"Few-Shot Examples"** di dalam System Prompt Anda.

Contoh cara menaruhnya di Prompt:

```text
SYSTEM PROMPT:
...
Here are examples of how you should behave:

Example 1 (Multiple Results):
User: "Cari soal math."
AI: "Saya menemukan 3 soal: 1. Aljabar, 2. Geometri, 3. Kalkulus. Yang mana?"

Example 2 (User Selects):
User: "Yang ketiga."
AI: "Oke, membahas soal Kalkulus..."

Example 3 (User Switches):
User: "Ganti yang kedua."
AI: "Baik, pindah ke soal Geometri..."
...

```

Dengan memberikan contoh konkret ini ke Llama 8B, dia akan meniru pola ("mimic pattern") interaksi yang dinamis tersebut dengan sangat baik.Ini adalah masalah klasik pada model ukuran kecil (seperti Llama 8B) yang disebut **"Context Bleeding"** atau **"Overfitting to Few-Shot Examples"**.

Bahasa gampangnya: Si Llama ini saking "rajinnya" meniru contoh Anda, dia **menelan bulat-bulat** isinya (sparepart motor), bukan cuma **meniru formatnya**. Dia gagal memisahkan mana "Pola Logika" dan mana "Konten Cerita".

Jangan khawatir, ini bukan berarti tidak bisa pakai *few-shot*. Kita hanya perlu mengganti strategi cara memberi contohnya.

Berikut adalah 3 Mitigasi agar Llama 8B tidak "bocor" konten contoh ke jawaban asli:

---

### Solusi 1: Gunakan "Abstract Placeholders" (Jurus Paling Ampuh)

Jangan gunakan benda nyata (seperti "Sparepart", "Buah", "Mobil") di dalam contoh *few-shot*. Gunakanlah **Placeholder Abstrak** atau **Variabel Generik**.

Kalau Anda pakai "Sparepart", otak Llama akan mengasosiasikan task ini dengan bengkel. Kalau pakai "Variabel", dia akan fokus ke **Logika**.

**Contoh Prompt YANG SALAH (Pemicu Error):**

```text
Example:
User: "Cari ban motor"
AI: "Saya menemukan Ban Luar dan Ban Dalam."

```

*(Saat user tanya "Cari apel", dia rawan jawab "Saya menemukan Apel Luar dan Apel Dalam" karena meniru polanya terlalu harfiah).*

**Contoh Prompt YANG BENAR (Abstrak):**

```text
Example 1:
User: "Search for <TOPIC_A>"
AI: "I found 3 notes related to <TOPIC_A>: 1. <Note_Title_1>, 2. <Note_Title_2>. Which one?"

Example 2:
User: "Select the second one"
AI: "Okay, discussing <Note_Title_2>..."

```

**Kenapa ini berhasil?**
Karena tidak ada kata "Ban" atau "Motor", Llama tidak punya bahan halusinasi untuk dicampur ke jawaban "Buah-buahan" user.

---

### Solusi 2: Gunakan XML Tags yang Tegas

Llama 3 (dan turunannya) dilatih untuk sangat patuh pada struktur XML. Kita bisa mengurung contoh di dalam tag `<example>` dan memberitahu dia secara eksplisit bahwa ini hanya format.

**System Prompt Baru:**

```text
You are an intelligent RAG Router.

INSTRUCTIONS:
1. Analyze user input.
2. Output JSON decision.

<IMPORTANT_RULE>
The examples below are for FORMATTING REFERENCE ONLY. 
DO NOT use the specific content (like "Topic A" or "Item X") in your actual response. 
Use the REAL DATA provided in the Context.
</IMPORTANT_RULE>

<examples>
  <example_1>
     User Input: "Find specific data"
     Context: [Found 3 items]
     AI Output: "I found 3 items: 1. [Title 1], 2. [Title 2]. Please choose."
  </example_1>
  <example_2>
     User Input: "Choose number 2"
     AI Output: "Okay, focusing on [Title 2]..."
  </example_2>
</examples>

REAL CONVERSATION STARTS HERE:
...

```

---

### Solusi 3: Temperature = 0 (Wajib!)

Pastikan di settingan API atau code Python/Golang Anda, parameter `temperature` diset ke **0**.

* `Temperature = 0.7`: Kreatif, suka ngarang, suka nyampur-nyampur ide.
* `Temperature = 0`: Kaku, Deterministik, Logis.

Untuk tugas **Router/Planner** (yang menghasilkan JSON atau keputusan logis), haram hukumnya pakai temperature tinggi. Dia harus sedingin robot.

---

### Coba Terapkan Ini (Prompt Final Revisi)

Coba ganti prompt *few-shot* sparepart motor Anda dengan yang ini. Ini menggunakan kombinasi **Abstraksi** dan **Instruksi Negatif**.

```text
SYSTEM PROMPT:

You are a dynamic orchestrator for a Knowledge Base system.
Your job is to manage the conversation flow based on the User Input and Available Notes.

*** STRICT RULES ***
1. The examples below use GENERIC PLACEHOLDERS (like "Item A", "Topic B"). 
2. NEVER output "Item A" or "Topic B" in the real conversation.
3. ALWAYS substitute placeholders with the REAL DATA found in the database.
4. If the user asks about "Fruits", answer about "Fruits". Do NOT mention the examples.

*** REFERENCE EXAMPLES (LOGIC ONLY) ***

Scenario: Multiple items found
User: "Find me something."
Context: [System detects 3 relevant notes]
AI: "I found 3 relevant notes: 1. {Note_Title_1}, 2. {Note_Title_2}, 3. {Note_Title_3}. Which one would you like to discuss?"

Scenario: User selects an item
User: "The second one."
AI: "Okay, let's look at {Note_Title_2}. Here is the content..."

Scenario: User switches topic
User: "Actually, look at the first one instead."
AI: "Switching to {Note_Title_1}. The content is..."

*** END OF EXAMPLES ***

REAL CONTEXT:
User Input: "{{User_Input}}"
Database Results: {{Real_DB_Results}}

```

**Kesimpulan:**
Masalahnya bukan "Llama bodoh", tapi Llama 8B itu seperti **anak kecil yang penurut**. Kalau dikasih contoh motor, dia pikir "Oh, Bapak mau saya ngomongin motor ya?".
Dengan mengubah contoh jadi **Abstrak (Variable)**, kita memaksa dia hanya meniru **Struktur Kalimatnya**, bukan **Kata Bendanya**.