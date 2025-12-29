package main

import (
	"encoding/json"
	"html/template"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type identitas struct {
	Nama      string
	NoAntrean int
}

type SearchResult struct {
	Metode    string  `json:"metode"`
	Nama      string  `json:"nama"`
	Waktu     float64 `json:"waktu"` // dalam mikrodetik
	Ditemukan bool    `json:"ditemukan"`
	Timestamp string  `json:"timestamp"`
	TotalData int     `json:"totalData"` // Total data saat pencarian
	Index     int     `json:"index"`     // Urutan pencarian
}

var (
	antrean       []identitas
	mu            sync.Mutex
	searchHistory []SearchResult
	searchIndex   int
	tmpl          = template.Must(template.New("index").Parse(htmlTemplate))
)

const htmlTemplate = `
<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="UTF-8">
<title>Antrian Kasir</title>
<script src="https://cdnjs.cloudflare.com/ajax/libs/Chart.js/3.9.1/chart.min.js"></script>
<style>
body {
	font-family: Arial, sans-serif;
	max-width: 1200px;
	margin: 0 auto;
	padding: 20px;
	background-color: #f5f5f5;
}
.container {
	background: white;
	padding: 30px;
	border-radius: 10px;
	box-shadow: 0 2px 10px rgba(0,0,0,0.1);
}
.section {
	margin-bottom: 30px;
	padding: 20px;
	background: #f9f9f9;
	border-radius: 8px;
}
h2 {
	color: #333;
	text-align: center;
	margin-bottom: 30px;
}
h3 {
	color: #555;
	border-bottom: 2px solid #4CAF50;
	padding-bottom: 10px;
}
input[type="text"], input[type="number"], select {
	padding: 8px;
	margin: 5px;
	border: 1px solid #ddd;
	border-radius: 4px;
	font-size: 14px;
}
input[type="submit"], button {
	background-color: #4CAF50;
	color: white;
	padding: 10px 20px;
	border: none;
	border-radius: 4px;
	cursor: pointer;
	font-size: 14px;
	margin: 5px;
}
input[type="submit"]:hover, button:hover {
	background-color: #45a049;
}
button.danger {
	background-color: #f44336;
}
button.danger:hover {
	background-color: #da190b;
}
ul {
	list-style: none;
	padding: 0;
}
li {
	padding: 8px;
	margin: 5px 0;
	background: white;
	border-left: 4px solid #4CAF50;
	border-radius: 4px;
}
.chart-container {
	position: relative;
	height: 400px;
	margin-top: 20px;
}
.generator-form {
	display: flex;
	align-items: center;
	gap: 10px;
	flex-wrap: wrap;
}
.info-box {
	background: #e3f2fd;
	padding: 15px;
	border-radius: 4px;
	margin: 10px 0;
	border-left: 4px solid #2196F3;
}
.stats-grid {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
	gap: 15px;
	margin-top: 15px;
}
.stat-card {
	background: white;
	padding: 15px;
	border-radius: 8px;
	border-left: 4px solid #4CAF50;
	box-shadow: 0 2px 5px rgba(0,0,0,0.1);
}
.stat-card h4 {
	margin: 0 0 5px 0;
	color: #666;
	font-size: 14px;
	font-weight: normal;
}
.stat-card .value {
	font-size: 24px;
	font-weight: bold;
	color: #333;
}
table {
	width: 100%;
	border-collapse: collapse;
}
th, td {
	padding: 10px;
	border: 1px solid #ddd;
	text-align: left;
}
th {
	background-color: #4CAF50;
	color: white;
}
tr:nth-child(even) {
	background-color: #f9f9f9;
}
</style>
</head>
<body>
<div class="container">
<h2>📋 Antrian Kasir - Sistem Pencarian</h2>

<div class="section">
<h3>🎲 Generate Data Otomatis</h3>
<form method="POST" action="/generate" class="generator-form">
<label>Jumlah Data:</label>
<input type="number" name="jumlah" min="1" value="100" required>
<input type="submit" value="Generate Data Random">
<button type="button" class="danger" onclick="if(confirm('Hapus semua data?')) document.getElementById('clearForm').submit();">Hapus Semua Data</button>
</form>
<form id="clearForm" method="POST" action="/clear" style="display:none;"></form>
<div class="info-box">
💡 <strong>Tips:</strong> Generate data untuk testing performa pencarian. Tidak ada batasan jumlah data!
</div>
</div>

<div class="section">
<h3>➕ Tambah Antrean Manual</h3>
<form method="POST" action="/tambah">
Nama: <input type="text" name="nama" required>
<input type="submit" value="Tambah">
</form>
</div>

<div class="section">
<h3>➖ Hapus Antrean</h3>
<form method="POST" action="/hapus">
<input type="submit" value="Hapus Antrean Pertama">
</form>
</div>

<div class="section">
<h3>🔍 Cari Antrean</h3>
<form method="GET" action="/cari">
Nama: <input type="text" name="nama" required>
Metode:
<select name="metode">
<option value="iteratif">Iteratif (Sequential)</option>
<option value="rekursif">Rekursif</option>
</select>
<input type="submit" value="Cari">
</form>

{{if .Cari}}
<h4>Hasil Pencarian:</h4>
<p>{{.Cari}}</p>
{{if .TimeValue}}
<p><strong>Waktu Eksekusi:</strong> {{printf "%.3f" .TimeValue}} mikrodetik ({{.TimeMicro}} ms)</p>
{{end}}
{{end}}

{{if gt .Jumlah 0}}
<div class="info-box">
💡 <strong>Saran Pencarian:</strong> Coba cari nama seperti: {{range $i, $v := .SampleNames}}{{if $i}}, {{end}}"{{$v}}"{{end}}
</div>
{{end}}
</div>

<div class="section">
<h3>📊 Grafik Running Time Pencarian (Sequential)</h3>
<button onclick="refreshGraph()">🔄 Refresh Grafik</button>
<button class="danger" onclick="clearHistory()">🗑️ Hapus Riwayat</button>
<div class="chart-container">
<canvas id="runtimeChart"></canvas>
</div>
<p id="chartMessage" style="text-align: center; color: #666; margin-top: 20px;"></p>

</div>

<div class="section">
<h3>📋 Tabel Data Pencarian</h3>
<div style="overflow-x: auto;">
<table id="searchTable" style="width: 100%; border-collapse: collapse; display: none;">
<thead>
<tr style="background-color: #4CAF50; color: white;">
<th style="padding: 12px; text-align: left; border: 1px solid #ddd;">No</th>
<th style="padding: 12px; text-align: left; border: 1px solid #ddd;">Nama</th>
<th style="padding: 12px; text-align: left; border: 1px solid #ddd;">Metode</th>
<th style="padding: 12px; text-align: right; border: 1px solid #ddd;">Waktu (μs)</th>
<th style="padding: 12px; text-align: center; border: 1px solid #ddd;">Status</th>
<th style="padding: 12px; text-align: center; border: 1px solid #ddd;">Total Data</th>
<th style="padding: 12px; text-align: left; border: 1px solid #ddd;">Timestamp</th>
</tr>
</thead>
<tbody id="searchTableBody">
</tbody>
</table>
<p id="tableMessage" style="text-align: center; color: #666; padding: 20px;">Belum ada data pencarian.</p>
</div>
</div>

<div class="section">
<h3>📝 Daftar Antrean (Total: {{.Jumlah}})</h3>
{{if eq .Jumlah 0}}
<p>❌ Tidak ada antrean.</p>
{{else}}
<ul>
{{range .Daftar}}
<li>{{.NoAntrean}}. {{.Nama}}</li>
{{end}}
</ul>
{{end}}
</div>

</div>

<script>
let chart = null;

function loadGraph(showMessage) {
    fetch('/api/search-history')
        .then(response => response.json())
        .then(data => {
            if (data === null || data.length === 0) {
                if (showMessage) {
                    document.getElementById('chartMessage').textContent = 'Belum ada data pencarian. Lakukan pencarian terlebih dahulu.';
                }
                if (chart) {
                    chart.destroy();
                    chart = null;
                }
                document.getElementById('searchTable').style.display = 'none';
                document.getElementById('tableMessage').style.display = 'block';
                return;
            }
            document.getElementById('chartMessage').textContent = '';
            renderChart(data);
            renderTable(data);
        })
        .catch(error => {
            console.error('Error:', error);
            document.getElementById('chartMessage').textContent = 'Gagal memuat data grafik.';
        });
}

function refreshGraph() {
    loadGraph(true);
}

function renderChart(data) {
    const ctx = document.getElementById('runtimeChart').getContext('2d');

    // Pisahkan data berdasarkan metode
    const iteratifData = [];
    const rekursifData = [];
    
    data.forEach(function(d) {
        if (d.metode === 'iteratif') {
            iteratifData.push({
                x: d.index,
                y: d.waktu,
                nama: d.nama,
                totalData: d.totalData
            });
        } else if (d.metode === 'rekursif') {
            rekursifData.push({
                x: d.index,
                y: d.waktu,
                nama: d.nama,
                totalData: d.totalData
            });
        }
    });

    // Hapus chart lama jika ada
    if (chart) {
        chart.destroy();
    }

    // Buat chart baru
    chart = new Chart(ctx, {
        type: 'line',
        data: {
            datasets: [
                {
                    label: 'Iteratif (μs)',
                    data: iteratifData,
                    borderColor: 'rgb(76, 175, 80)',
                    backgroundColor: 'rgba(76, 175, 80, 0.1)',
                    borderWidth: 3,
                    tension: 0.4,
                    pointRadius: 5,
                    pointHoverRadius: 7
                },
                {
                    label: 'Rekursif (μs)',
                    data: rekursifData,
                    borderColor: 'rgb(244, 67, 54)',
                    backgroundColor: 'rgba(244, 67, 54, 0.1)',
                    borderWidth: 3,
                    tension: 0.4,
                    pointRadius: 5,
                    pointHoverRadius: 7
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                title: {
                    display: true,
                    text: 'Perbandingan Running Time Iteratif vs Rekursif (Sequential)',
                    font: {
                        size: 16
                    }
                },
                legend: {
                    display: true,
                    position: 'top'
                },
                tooltip: {
                    callbacks: {
                        label: function(context) {
                            const point = context.raw;
                            return [
                                context.dataset.label + ': ' + point.y.toFixed(3) + ' μs',
                                'Nama: ' + point.nama,
                                'Total Data: ' + point.totalData
                            ];
                        }
                    }
                }
            },
            scales: {
                y: {
                    beginAtZero: true,
                    title: {
                        display: true,
                        text: 'Waktu (mikrodetik)'
                    }
                },
                x: {
                    type: 'linear',
                    title: {
                        display: true,
                        text: 'Urutan Pencarian'
                    },
                    ticks: {
                        stepSize: 1
                    }
                }
            }
        }
    });
}

function renderTable(data) {
    const tbody = document.getElementById('searchTableBody');
    tbody.innerHTML = '';
    
    data.forEach((item, index) => {
        const row = document.createElement('tr');
        row.style.backgroundColor = index % 2 === 0 ? '#f9f9f9' : 'white';
        
        const statusIcon = item.ditemukan ? '✓' : '✗';
        const statusColor = item.ditemukan ? '#4CAF50' : '#f44336';
        const metodeLabel = item.metode === 'iteratif' ? 'Iteratif' : 'Rekursif';
        
        row.innerHTML = '<td style="padding: 10px; border: 1px solid #ddd;">' + (index + 1) + '</td>' +
            '<td style="padding: 10px; border: 1px solid #ddd;">' + item.nama + '</td>' +
            '<td style="padding: 10px; border: 1px solid #ddd;">' + metodeLabel + '</td>' +
            '<td style="padding: 10px; border: 1px solid #ddd; text-align: right;">' + item.waktu.toFixed(3) + '</td>' +
            '<td style="padding: 10px; border: 1px solid #ddd; text-align: center; color: ' + statusColor + '; font-weight: bold;">' + statusIcon + '</td>' +
            '<td style="padding: 10px; border: 1px solid #ddd; text-align: center; font-weight: bold;">' + item.totalData.toLocaleString() + '</td>' +
            '<td style="padding: 10px; border: 1px solid #ddd;">' + item.timestamp + '</td>';
        
        tbody.appendChild(row);
    });
    
    document.getElementById('searchTable').style.display = 'table';
    document.getElementById('tableMessage').style.display = 'none';
}

function clearHistory() {
    if (confirm('Yakin ingin menghapus semua riwayat pencarian?')) {
        fetch('/api/clear-history', {method: 'POST'})
            .then(() => {
                if (chart) {
                    chart.destroy();
                    chart = null;
                }
                document.getElementById('chartMessage').textContent = 'Riwayat berhasil dihapus.';
                document.getElementById('searchTable').style.display = 'none';
                document.getElementById('tableMessage').style.display = 'block';
                setTimeout(() => {
                    location.reload();
                }, 1000);
            })
            .catch(error => {
                console.error('Error:', error);
                alert('Gagal menghapus riwayat');
            });
    }
}

window.onload = function() {
    loadGraph(false);
};
</script>
</body>
</html>
`

type TemplateData struct {
	Jumlah      int
	Daftar      []identitas
	Cari        string
	TimeValue   float64
	TimeMicro   string
	SampleNames []string
}

func main() {
	rand.Seed(time.Now().UnixNano())
	antrean = make([]identitas, 0)
	searchIndex = 0

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/tambah", tambahHandler)
	http.HandleFunc("/hapus", hapusHandler)
	http.HandleFunc("/cari", cariHandler)
	http.HandleFunc("/generate", generateHandler)
	http.HandleFunc("/clear", clearHandler)
	http.HandleFunc("/api/search-history", searchHistoryHandler)
	http.HandleFunc("/api/clear-history", clearHistoryHandler)

	println("Server berjalan di http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

// ================= HANDLER =================

func indexHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	sampleNames := getSampleNames(antrean, 5)

	data := TemplateData{
		Jumlah:      len(antrean),
		Daftar:      tampilkanAntrean(antrean),
		SampleNames: sampleNames,
	}
	tmpl.Execute(w, data)
}

func tambahHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	nama := r.FormValue("nama")
	mu.Lock()
	tambahAntrean(nama)
	mu.Unlock()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func hapusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	mu.Lock()
	hapusAntrean()
	mu.Unlock()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func generateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	jumlahStr := r.FormValue("jumlah")
	jumlahGenerate, err := strconv.Atoi(jumlahStr)
	if err != nil || jumlahGenerate < 1 {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	mu.Lock()
	generateRandomData(jumlahGenerate)
	mu.Unlock()

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func clearHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	mu.Lock()
	antrean = make([]identitas, 0)
	mu.Unlock()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func cariHandler(w http.ResponseWriter, r *http.Request) {
	nama := r.URL.Query().Get("nama")
	metode := r.URL.Query().Get("metode")

	mu.Lock()
	totalDataSaatIni := len(antrean)
	hasil, waktuMicro, ditemukan := cariAntreanDenganWaktu(antrean, nama, metode)

	searchIndex++
	searchHistory = append(searchHistory, SearchResult{
		Metode:    metode,
		Nama:      nama,
		Waktu:     waktuMicro,
		Ditemukan: ditemukan,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		TotalData: totalDataSaatIni,
		Index:     searchIndex,
	})

	sampleNames := getSampleNames(antrean, 5)
	jumlahData := len(antrean)
	daftarData := tampilkanAntrean(antrean)
	mu.Unlock()

	waktuMs := strconv.FormatFloat(waktuMicro/1000.0, 'f', 3, 64)

	data := TemplateData{
		Jumlah:      jumlahData,
		Daftar:      daftarData,
		Cari:        hasil,
		TimeValue:   waktuMicro,
		TimeMicro:   waktuMs,
		SampleNames: sampleNames,
	}
	tmpl.Execute(w, data)
}

func searchHistoryHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if searchHistory == nil {
		json.NewEncoder(w).Encode([]SearchResult{})
	} else {
		json.NewEncoder(w).Encode(searchHistory)
	}
}

func clearHistoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mu.Lock()
	searchHistory = []SearchResult{}
	searchIndex = 0
	mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

// ================= FUNGSI GENERATOR =================

var namaDepan = []string{
	"Ahmad", "Budi", "Citra", "Dewi", "Eko", "Farah", "Gilang", "Hani",
	"Indra", "Joko", "Kartika", "Lina", "Maya", "Nanda", "Omar", "Putri",
	"Qori", "Rini", "Sari", "Tono", "Umar", "Vina", "Wati", "Ximena",
	"Yuni", "Zahra", "Agus", "Bella", "Clara", "Doni", "Elsa", "Fandi",
}

var namaBelakang = []string{
	"Pratama", "Wibowo", "Kusuma", "Santoso", "Wijaya", "Setiawan", "Permana",
	"Hidayat", "Rahman", "Nugroho", "Saputra", "Ramadan", "Hakim", "Putra",
	"Sanjaya", "Mahendra", "Purnama", "Adiputra", "Firmansyah", "Kurniawan",
}

func generateRandomData(count int) {
	antrean = make([]identitas, 0, count)

	for i := 0; i < count; i++ {
		depan := namaDepan[rand.Intn(len(namaDepan))]
		belakang := namaBelakang[rand.Intn(len(namaBelakang))]
		nama := depan + " " + belakang

		if rand.Float32() < 0.3 {
			nama += " " + strconv.Itoa(rand.Intn(100))
		}

		tambahAntrean(nama)
	}
}

func getSampleNames(daftar []identitas, count int) []string {
	jumlah := len(daftar)
	if jumlah == 0 {
		return []string{}
	}

	samples := []string{}
	step := 1
	if jumlah > count {
		step = jumlah / count
	}

	for i := 0; i < jumlah && len(samples) < count; i += step {
		samples = append(samples, daftar[i].Nama)
	}

	return samples
}

// ================= FUNGSI ASLI =================

func tambahAntrean(nama string) {
	noAntrean := len(antrean) + 1
	antrean = append(antrean, identitas{
		Nama:      nama,
		NoAntrean: noAntrean,
	})
}

func hapusAntrean() {
	if len(antrean) == 0 {
		return
	}
	antrean = antrean[1:]
	for i := 0; i < len(antrean); i++ {
		antrean[i].NoAntrean = i + 1
	}
}

func tampilkanAntrean(daftar []identitas) []identitas {
	if len(daftar) == 0 {
		return nil
	}
	return daftar
}

func searchIteratif(daftar []identitas, nama string) int {	
	for i := 0; i < len(daftar); i++ {
		if daftar[i].Nama == nama {
			return i
		}
	}
	return -1
}

	func searchRekursif(daftar []identitas, jumlah int, nama string) int {
		if jumlah == 0 {
			return -1
		} else {
			if daftar[jumlah-1].Nama == nama {
				return jumlah - 1
			} else {
				return searchRekursif(daftar, jumlah-1, nama)
			}
		}
	}

func cariAntreanDenganWaktu(daftar []identitas, nama string, metode string) (string, float64, bool) {
	jumlah := len(daftar)
	if jumlah == 0 {
		return "❌ Antrean kosong, tidak bisa mencari.", 0, false
	}

	var posisi int
	var start time.Time
	var duration time.Duration

	if metode == "iteratif" {
		start = time.Now()
		posisi = searchIteratif(daftar, nama)
		duration = time.Since(start)
	} else {
		start = time.Now()
		posisi = searchRekursif(daftar, jumlah, nama)
		duration = time.Since(start)
	}

	waktuMicro := float64(duration.Nanoseconds()) / 1000.0

	if posisi != -1 {
		hasil := "✔ Pelanggan ditemukan! Nama: " + daftar[posisi].Nama + " | No Antrean: " + strconv.Itoa(daftar[posisi].NoAntrean)
		return hasil, waktuMicro, true
	}
	return "❌ Pelanggan tidak ditemukan.", waktuMicro, false
}