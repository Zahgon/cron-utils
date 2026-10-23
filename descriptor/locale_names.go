// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package descriptor

// displayNames holds the full (format-context) month and weekday names of the Unicode CLDR locale data for the
// languages of the bundled description texts; the empty key is the root locale.
type displayNames struct {
	months [12]string
	days   [7]string // Monday first
}

var localeNames = map[string]displayNames{
	"de": {
		months: [12]string{"Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"},
		days:   [7]string{"Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag", "Sonntag"},
	},
	"el": {
		months: [12]string{"Ιανουαρίου", "Φεβρουαρίου", "Μαρτίου", "Απριλίου", "Μαΐου", "Ιουνίου", "Ιουλίου", "Αυγούστου", "Σεπτεμβρίου", "Οκτωβρίου", "Νοεμβρίου", "Δεκεμβρίου"},
		days:   [7]string{"Δευτέρα", "Τρίτη", "Τετάρτη", "Πέμπτη", "Παρασκευή", "Σάββατο", "Κυριακή"},
	},
	"en": {
		months: [12]string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"},
		days:   [7]string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"},
	},
	"es": {
		months: [12]string{"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"},
		days:   [7]string{"lunes", "martes", "miércoles", "jueves", "viernes", "sábado", "domingo"},
	},
	"fr": {
		months: [12]string{"janvier", "février", "mars", "avril", "mai", "juin", "juillet", "août", "septembre", "octobre", "novembre", "décembre"},
		days:   [7]string{"lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi", "dimanche"},
	},
	"id": {
		months: [12]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"},
		days:   [7]string{"Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu", "Minggu"},
	},
	"it": {
		months: [12]string{"gennaio", "febbraio", "marzo", "aprile", "maggio", "giugno", "luglio", "agosto", "settembre", "ottobre", "novembre", "dicembre"},
		days:   [7]string{"lunedì", "martedì", "mercoledì", "giovedì", "venerdì", "sabato", "domenica"},
	},
	"ja": {
		months: [12]string{"1月", "2月", "3月", "4月", "5月", "6月", "7月", "8月", "9月", "10月", "11月", "12月"},
		days:   [7]string{"月曜日", "火曜日", "水曜日", "木曜日", "金曜日", "土曜日", "日曜日"},
	},
	"ko": {
		months: [12]string{"1월", "2월", "3월", "4월", "5월", "6월", "7월", "8월", "9월", "10월", "11월", "12월"},
		days:   [7]string{"월요일", "화요일", "수요일", "목요일", "금요일", "토요일", "일요일"},
	},
	"nl": {
		months: [12]string{"januari", "februari", "maart", "april", "mei", "juni", "juli", "augustus", "september", "oktober", "november", "december"},
		days:   [7]string{"maandag", "dinsdag", "woensdag", "donderdag", "vrijdag", "zaterdag", "zondag"},
	},
	"pl": {
		months: [12]string{"stycznia", "lutego", "marca", "kwietnia", "maja", "czerwca", "lipca", "sierpnia", "września", "października", "listopada", "grudnia"},
		days:   [7]string{"poniedziałek", "wtorek", "środa", "czwartek", "piątek", "sobota", "niedziela"},
	},
	"pt": {
		months: [12]string{"janeiro", "fevereiro", "março", "abril", "maio", "junho", "julho", "agosto", "setembro", "outubro", "novembro", "dezembro"},
		days:   [7]string{"segunda-feira", "terça-feira", "quarta-feira", "quinta-feira", "sexta-feira", "sábado", "domingo"},
	},
	"ro": {
		months: [12]string{"ianuarie", "februarie", "martie", "aprilie", "mai", "iunie", "iulie", "august", "septembrie", "octombrie", "noiembrie", "decembrie"},
		days:   [7]string{"luni", "marți", "miercuri", "joi", "vineri", "sâmbătă", "duminică"},
	},
	"ru": {
		months: [12]string{"января", "февраля", "марта", "апреля", "мая", "июня", "июля", "августа", "сентября", "октября", "ноября", "декабря"},
		days:   [7]string{"понедельник", "вторник", "среда", "четверг", "пятница", "суббота", "воскресенье"},
	},
	"sw": {
		months: [12]string{"Januari", "Februari", "Machi", "Aprili", "Mei", "Juni", "Julai", "Agosti", "Septemba", "Oktoba", "Novemba", "Desemba"},
		days:   [7]string{"Jumatatu", "Jumanne", "Jumatano", "Alhamisi", "Ijumaa", "Jumamosi", "Jumapili"},
	},
	"tr": {
		months: [12]string{"Ocak", "Şubat", "Mart", "Nisan", "Mayıs", "Haziran", "Temmuz", "Ağustos", "Eylül", "Ekim", "Kasım", "Aralık"},
		days:   [7]string{"Pazartesi", "Salı", "Çarşamba", "Perşembe", "Cuma", "Cumartesi", "Pazar"},
	},
	"zh": {
		months: [12]string{"一月", "二月", "三月", "四月", "五月", "六月", "七月", "八月", "九月", "十月", "十一月", "十二月"},
		days:   [7]string{"星期一", "星期二", "星期三", "星期四", "星期五", "星期六", "星期日"},
	},
	"": {
		months: [12]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"},
		days:   [7]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
	},
}
