/**
 * Вспомогательные функции, вынесенные для модульного тестирования
 */
function parseTimeToMinutes(t) {
  if (!t || typeof t !== 'string' || !t.includes(':')) {
    throw new Error('Invalid time format');
  }
  const [hh, mm] = t.split(':').map(Number);
  return hh * 60 + mm;
}

function nextHourLabel(timeStr) {
  const m = parseTimeToMinutes(timeStr);
  const next = m + 60;
  const hh = Math.floor(next / 60) % 24;
  const mm = next % 60;
  return `${String(hh).padStart(2, '0')}:${String(mm).padStart(2, '0')}`;
}
function resetState() {
  wsBooking = null;
  wsConnected = false;
  serverDatesMap = {};
  availableDatesSet = new Set();
  selectedDate = null;
  selectedSlots = [];
  calendarDays = [];
}

function formatYMD(d) {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

function weekdayRu(d) {
  return ['Вс', 'Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб'][d.getDay()];
}

function formatDayLabel(d) {
  return `${d.getDate()}.${String(d.getMonth() + 1).padStart(2, '0')}`;
}

function nextDay(d, n = 1) {
  const x = new Date(d);
  x.setDate(x.getDate() + n);
  return x;
}
function navigate(url) {
  window.location.href = url;
}

// --- ПЕРЕМЕННЫЕ СОСТОЯНИЯ (вынесены для работы модульных тестов) ---
let wsBooking = null;
let wsConnected = false;
let serverDatesMap = {};
let availableDatesSet = new Set();
let selectedDate = null;
let selectedSlots = [];
let calendarDays = [];

// --- ОСНОВНЫЕ ФУНКЦИИ БРОНИРОВАНИЯ И АВТОРИЗАЦИИ (вынесены для Jest) ---

function detectAuthEarly() {
  const authButtonEarly = document.getElementById('authButton');
  if (authButtonEarly && authButtonEarly.dataset && typeof authButtonEarly.dataset.auth !== 'undefined') {
    return authButtonEarly.dataset.auth === '1';
  }
  if (window.__USER__ && (window.__USER__.auth || window.__USER__.username)) {
    return !!window.__USER__.auth || !!window.__USER__.username;
  }
  return false;
}

function readServerAuth() {
  const authButton = document.getElementById('authButton') || document.querySelector('.auth-btn');
  if (authButton && authButton.dataset && typeof authButton.dataset.auth !== 'undefined') {
    return { auth: authButton.dataset.auth === '1', username: authButton.dataset.username || null };
  }
  if (window.__USER__) return { auth: !!window.__USER__.auth, username: window.__USER__.username || null };
  return { auth: false, username: null };
}

function showBookHint(show) {
  const bookBtn = document.getElementById('bookBtn');
  if (!bookBtn) return;
  const parent = bookBtn.parentNode || bookBtn.parentElement;
  if (!parent) return;
  let hint = document.getElementById('bookHint');
  if (show) {
    if (!hint) {
      hint = document.createElement('div');
      hint.id = 'bookHint';
      hint.className = 'book-hint';
      hint.textContent = 'Вам необходимо войти в систему для бронирования';
      parent.appendChild(hint);
    }
  } else {
    if (hint) hint.remove();
  }
}

function buildCalendarDays() {
  calendarDays = [];
  const today = new Date();
  for (let i = 0; i < 7; i++) {
    const d = nextDay(today, i);
    calendarDays.push({ ymd: formatYMD(d), dateObj: d });
  }
}

function updateSummary() {
  const selectionSummary = document.getElementById('selectionSummary');
  if (!selectionSummary) return;
  if (!selectedDate || selectedSlots.length === 0) selectionSummary.textContent = 'Слоты: не выбраны';
  else {
    const ord = [...selectedSlots].sort((a, b) => parseTimeToMinutes(a) - parseTimeToMinutes(b));
    selectionSummary.textContent = `Дата: ${selectedDate}, слоты: ${ord.join(', ')}`;
  }
}

function renderCalendar() {
  const bookingCalendar = document.getElementById('bookingCalendar');
  if (!bookingCalendar) return;
  bookingCalendar.innerHTML = '';
  for (const day of calendarDays) {
    const div = document.createElement('button');
    div.type = 'button';
    div.className = 'day';
    div.dataset.ymd = day.ymd;
    const dLabel = document.createElement('span'); dLabel.className = 'd-date'; dLabel.textContent = formatDayLabel(day.dateObj);
    const wLabel = document.createElement('span'); wLabel.className = 'd-weekday'; wLabel.textContent = weekdayRu(day.dateObj);
    div.appendChild(dLabel); div.appendChild(wLabel);

    if (availableDatesSet.has(day.ymd)) div.classList.remove('disabled');
    else div.classList.add('disabled');

    if (selectedDate === day.ymd) div.classList.add('selected');
    else div.classList.remove('selected');

    div.addEventListener('click', () => {
      if (selectedDate === day.ymd) return;
      selectedDate = day.ymd;
      selectedSlots = [];
      renderCalendar();
      renderTimes(serverDatesMap[day.ymd] || []);
      updateSummary();
    });
    bookingCalendar.appendChild(div);
  }
}

function renderTimes(timesArray) {
  const bookingTimes = document.getElementById('bookingTimes');
  if (!bookingTimes) return;
  bookingTimes.innerHTML = '';
  if (!selectedDate) {
    bookingTimes.innerHTML = '<div style="color:var(--muted)">Выберите дату</div>';
    return;
  }
  if (!timesArray || timesArray.length === 0) {
    bookingTimes.innerHTML = '<div style="color:var(--muted)">Нет доступных слотов</div>';
    return;
  }
  const times = [...timesArray].sort((a, b) => parseTimeToMinutes(a) - parseTimeToMinutes(b));
  for (const t of times) {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'slot';
    btn.dataset.time = t;
    btn.textContent = `${t} — ${nextHourLabel(t)}`;
    if (selectedSlots.includes(t)) btn.classList.add('selected');
    btn.addEventListener('click', () => onSlotClick(t));
    bookingTimes.appendChild(btn);
  }
}

function onSlotClick(time) {
  const idx = selectedSlots.indexOf(time);
  if (idx !== -1) {
    if (selectedSlots.length === 1) selectedSlots = [];
    else {
      if (time === selectedSlots[0]) selectedSlots.shift();
      else if (time === selectedSlots[selectedSlots.length - 1]) selectedSlots.pop();
      else selectedSlots = [time];
    }
    renderTimes(serverDatesMap[selectedDate] || []);
    updateSummary();
    return;
  }
  if (selectedSlots.length === 0) selectedSlots = [time];
  else {
    const sorted = [...selectedSlots].sort((a, b) => parseTimeToMinutes(a) - parseTimeToMinutes(b));
    const first = sorted[0];
    const last = sorted[sorted.length - 1];
    if (parseTimeToMinutes(time) + 60 === parseTimeToMinutes(first)) selectedSlots = [time, ...sorted];
    else if (parseTimeToMinutes(time) === parseTimeToMinutes(last) + 60) selectedSlots = [...sorted, time];
    else selectedSlots = [time];
  }
  renderTimes(serverDatesMap[selectedDate] || []);
  updateSummary();
}

function getBookingWsUrl() {
  const scheme = (window.location.protocol === 'https:') ? 'wss:' : 'ws:';
  return `${scheme}//${window.location.host}/ws/booking`;
}

function openWsBooking() {
  if (wsBooking && wsConnected) return;
  serverDatesMap = {};
  availableDatesSet = new Set();
  try {
    wsBooking = new WebSocket(getBookingWsUrl());
  } catch (err) {
    console.error('ws booking create failed', err);
    return;
  }
  
  const R = window.__ROOM__ || {};
  
  wsBooking.addEventListener('open', () => {
    wsConnected = true;
    wsBooking.send(JSON.stringify({ room_id: Number(R.id) }));
  });
  
  wsBooking.addEventListener('message', ev => {
    try {
      const data = JSON.parse(ev.data);
      if (data && Array.isArray(data.dates)) {
        serverDatesMap = {};
        availableDatesSet = new Set();
        data.dates.forEach(d => {
          if (d && d.date && Array.isArray(d.available_times)) {
            serverDatesMap[d.date] = d.available_times;
            availableDatesSet.add(d.date);
          }
        });
        if (!selectedDate || !serverDatesMap[selectedDate]) {
          const first = calendarDays.find(cd => availableDatesSet.has(cd.ymd));
          if (first) selectedDate = first.ymd;
        }
        renderCalendar();
        renderTimes(serverDatesMap[selectedDate] || []);
        
        const bookingNotice = document.getElementById('bookingNotice');
        if (bookingNotice) bookingNotice.textContent = 'Выберите дату и слоты';
      }
    } catch (err) { console.warn('parse ws booking', err); }
  });
  wsBooking.addEventListener('close', () => { wsConnected = false; wsBooking = null; });
}

function closeWsBooking() {
  if (wsBooking) wsBooking.close();
  wsConnected = false;
}

function openBookingModal() {
  const isAuthNow = detectAuthEarly();
  if (!isAuthNow) {
    module.exports.navigate('/auth');
    return;
  }
  const bookingModal = document.getElementById('bookingModal');
  if (!bookingModal) return;
  
  buildCalendarDays();
  renderCalendar();
  openWsBooking();
  
  bookingModal.classList.remove('hidden');
  bookingModal.setAttribute('aria-hidden', 'false');
  document.body.style.overflow = 'hidden';
  selectedDate = null;
  selectedSlots = [];
  renderTimes([]);
  updateSummary();
  
  const bookingNotice = document.getElementById('bookingNotice');
  if (bookingNotice) bookingNotice.textContent = '';
}

function closeBookingModal() {
  const bookingModal = document.getElementById('bookingModal');
  if (!bookingModal) return;
  bookingModal.classList.add('hidden');
  bookingModal.setAttribute('aria-hidden', 'true');
  document.body.style.overflow = '';
  closeWsBooking();
}

async function confirmBooking() {
  const R = window.__ROOM__ || {};
  const bookingConfirm = document.getElementById('bookingConfirm');
  const bookingModal = document.getElementById('bookingModal');

  if (!selectedDate || selectedSlots.length === 0) { alert('Выберите дату и хотя бы один слот'); return; }
  const ord = [...selectedSlots].sort((a, b) => parseTimeToMinutes(a) - parseTimeToMinutes(b));
  for (let i = 1; i < ord.length; i++) {
    if (parseTimeToMinutes(ord[i]) !== parseTimeToMinutes(ord[i - 1]) + 60) {
      alert('Слоты выбраны не подряд.'); return;
    }
  }
  const payload = { room_id: Number(R.id), date: selectedDate, slots: ord };
  if (bookingConfirm) { bookingConfirm.disabled = true; bookingConfirm.textContent = 'Отправка...'; }
  try {
    const res = await fetch('/api/booking/new', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'same-origin',
      body: JSON.stringify(payload)
    });
    if (res.ok) {
      if (bookingModal) {
        bookingModal.querySelector('.booking-modal__content').innerHTML = `
          <h3>Бронирование успешно</h3>
          <div style="display:flex;gap:8px;margin-top:14px">
            <a href="/" class="link-as-btn primary">На главную</a>
            <a href="/me" class="link-as-btn">В кабинет</a>
          </div>`;
      }
      closeWsBooking();
    } else { alert('Ошибка бронирования'); }
  } catch (err) { alert('Ошибка соединения'); }
  finally { if (bookingConfirm) { bookingConfirm.disabled = false; bookingConfirm.textContent = 'Подтвердить'; } }
}

/**
 * Основная инициализация (остались только привязки событий)
 */
document.addEventListener('DOMContentLoaded', () => {
  const R = window.__ROOM__ || {};
  
  // 1. Галерея
  const mainImg = document.getElementById('mainImage');
  const thumbs = document.getElementById('thumbs');
  if (thumbs && mainImg) {
    thumbs.addEventListener('click', (e) => {
      const btn = e.target.closest('.thumb-btn');
      if (!btn) return;
      const src = btn.dataset.src || (btn.querySelector('img') && btn.querySelector('img').src);
      if (!src) return;
      mainImg.src = src;
    });
  }

  // 2. Карта
  const lat = (typeof R.lat === 'number') ? R.lat : (R.lat ? parseFloat(R.lat) : null);
  const lng = (typeof R.lng === 'number') ? R.lng : (R.lng ? parseFloat(R.lng) : null);
  if (lat && lng) {
    try {
      const map = L.map('map', { scrollWheelZoom: false }).setView([lat, lng], 15);
      map.attributionControl.setPrefix('');
      L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        maxZoom: 19,
        attribution: '&copy; OpenStreetMap contributors'
      }).addTo(map);
      L.marker([lat, lng]).addTo(map);
    } catch (err) {
      console.warn('Leaflet init failed', err);
    }
  } else {
    const mapWrap = document.getElementById('map');
    if (mapWrap) mapWrap.innerHTML = '<div style="padding:18px;color:var(--muted)">Координаты отсутствуют</div>';
  }

  // 3. Состояние кнопки бронирования при загрузке
  const bookBtn = document.getElementById('bookBtn');
  const initiallyAuthenticated = detectAuthEarly();

  if (!R.id) {
    if (bookBtn) bookBtn.disabled = true;
  }

  if (bookBtn) {
    if (!initiallyAuthenticated) {
      bookBtn.disabled = true;
      showBookHint(true);
    } else {
      bookBtn.disabled = false;
      showBookHint(false);
    }
    
    bookBtn.addEventListener('click', (e) => {
      e.preventDefault();
      if (bookBtn.disabled) { navigate('/auth'); return; }
      openBookingModal();
    });
  }

  // 4. Привязка кнопок модального окна
  const bookingClose = document.getElementById('bookingClose');
  const bookingCancel = document.getElementById('bookingCancel');
  const bookingConfirmBtn = document.getElementById('bookingConfirm');
  const bookingModal = document.getElementById('bookingModal');

  if (bookingClose) bookingClose.addEventListener('click', closeBookingModal);
  if (bookingCancel) bookingCancel.addEventListener('click', closeBookingModal);
  if (bookingConfirmBtn) bookingConfirmBtn.addEventListener('click', confirmBooking);

  if (bookingModal) {
    bookingModal.addEventListener('click', (e) => {
      if (e.target === bookingModal) closeBookingModal();
    });
  }

  // 5. Кнопки авторизации
  const authButton = document.getElementById('authButton') || document.querySelector('.auth-btn');
  const logoutBtn = document.getElementById('logoutBtn');
  let isAuthenticated = !!readServerAuth().auth;

  if (authButton) {
    authButton.textContent = isAuthenticated ? 'Кабинет' : 'Войти';
    authButton.addEventListener('click', (e) => {
      e.stopPropagation();
      navigate(isAuthenticated ? '/me' : '/auth');
    });
  }

  if (logoutBtn) {
    logoutBtn.addEventListener('click', async () => {
      try {
        const res = await fetch('/api/logout', { method: 'POST', credentials: 'same-origin' });
        if (res.ok) window.location.reload();
      } catch (err) { alert('Ошибка соединения'); }
    });
  }
});

// Блок экспорта для модульных тестов (Jest)
if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    parseTimeToMinutes, nextHourLabel, formatYMD,
    weekdayRu, formatDayLabel, nextDay,
    renderCalendar, renderTimes, onSlotClick, confirmBooking,
    buildCalendarDays, updateSummary, closeWsBooking,
    // Добавленные из DOMContentLoaded
    detectAuthEarly, readServerAuth, showBookHint, 
    openBookingModal, closeBookingModal, getBookingWsUrl, openWsBooking, navigate, resetState
  };
}