(function () {
  'use strict';

  let currentRoom = null;
  let currentUser = null;
  let mapInstance = null;

  let serverDatesMap = {};
  let availableDatesSet = new Set();
  let selectedDate = null;
  let selectedSlots = [];
  let calendarDays = [];

  document.addEventListener('DOMContentLoaded', initRoomPage);

  async function initRoomPage() {
    const roomId = getRoomIdFromUrl();

    bindHeaderActions();
    bindBookingModalActions();

    if (!roomId) {
      renderRoomError('Некорректный адрес страницы комнаты');
      return;
    }

    try {
      await loadCurrentUser();
      updateAuthUi();

      const data = await window.Api.getRoom(roomId);
      currentRoom = normalizeRoom(data);

      renderRoom(currentRoom);
      configureBookingButton();
    } catch (err) {
      console.error(err);
      renderRoomError(err?.message || 'Не удалось загрузить комнату');
    }
  }

  function getRoomIdFromUrl() {
    const match = window.location.pathname.match(/^\/room\/(\d+)$/);

    if (match) {
      return Number(match[1]);
    }

    const fromDataset = Number(document.body.dataset.roomId || 0);

    return Number.isFinite(fromDataset) && fromDataset > 0 ? fromDataset : null;
  }

  async function loadCurrentUser() {
    try {
      currentUser = await window.Api.getMe();
    } catch (err) {
      console.warn('Не удалось получить текущего пользователя:', err);
      currentUser = {
        authenticated: false,
        user: null
      };
    }
  }

  function isAuthenticated() {
    return !!currentUser?.authenticated;
  }

  function updateAuthUi() {
    const authButton = document.getElementById('authButton') || document.querySelector('.auth-btn');
    const logoutBtn = document.getElementById('logoutBtn');
    const historyBtn = document.getElementById('historyBtn');
    const userGreeting = document.getElementById('userGreeting');

    if (authButton) {
      authButton.dataset.auth = isAuthenticated() ? '1' : '0';
      authButton.textContent = isAuthenticated() ? 'Кабинет' : 'Войти';

      if (currentUser?.user?.username) {
        authButton.dataset.username = currentUser.user.username;
      }

      authButton.addEventListener('click', (e) => {
        e.stopPropagation();
        navigate(isAuthenticated() ? '/me' : '/auth');
      });
    }

    if (userGreeting && currentUser?.user?.username) {
      userGreeting.textContent = `Привет, ${currentUser.user.username}`;
    }

    if (historyBtn) {
      historyBtn.addEventListener('click', () => navigate('/me'));
    }

    if (logoutBtn) {
      logoutBtn.addEventListener('click', async () => {
        try {
          await window.Api.logoutUser();
          window.location.reload();
        } catch (err) {
          alert(err?.message || 'Ошибка выхода');
        }
      });
    }
  }

  function bindHeaderActions() {
    const brand = document.querySelector('.brand');

    if (brand) {
      brand.addEventListener('click', (e) => {
        e.preventDefault();
        navigate('/');
      });
    }
  }

  function normalizeRoom(data) {
    const room = data?.room || data || {};

    return {
      id: Number(room.id),
      title: room.title || room.name || 'Без названия',
      company: room.company || room.company_name || '',
      address: room.address || '',
      images: Array.isArray(room.images) ? room.images : [],
      price: room.price ?? 0,
      currency: room.currency || 'RUB',
      capacity: room.capacity ?? room.max_capacity ?? 0,
      max_capacity: room.max_capacity ?? room.capacity ?? 0,
      available_from: room.available_from || room.availableFrom || '',
      available_to: room.available_to || room.availableTo || '',
      description: room.description || '',
      lat: toNumberOrNull(room.lat ?? room.latitude),
      lng: toNumberOrNull(room.lng ?? room.longitude)
    };
  }

  function renderRoom(room) {
    document.body.dataset.roomId = String(room.id);
    document.title = `${room.title} — ${room.company || 'Комната'}`;

    setText('.room-info h1', room.title);
    setText('.company-name', room.company);
    setText('.room-info .address', room.address);
    setText('.price', `${room.price || 0} ₽/ч`);
    setText('.capacity', `Вместимость: ${room.capacity || 0} чел.`);
    setText('.availability .value', `c ${room.available_from || '—'} до ${room.available_to || '—'}`);
    setText('#description', room.description || 'Описание отсутствует');

    renderImages(room);
    renderDetails(room);
    renderMap(room);
  }

  function renderImages(room) {
    const mainImg = document.getElementById('mainImage');
    const thumbs = document.getElementById('thumbs');

    const images = room.images.length > 0
      ? room.images
      : ['/shared/placeholders/room-placeholder.svg'];

    if (mainImg) {
      mainImg.src = images[0];
      mainImg.alt = room.title;
    }

    if (!thumbs) return;

    thumbs.innerHTML = '';

    images.forEach((src, index) => {
      const btn = document.createElement('button');
      btn.className = 'thumb-btn';
      btn.type = 'button';
      btn.dataset.src = src;
      btn.setAttribute('aria-label', `Изображение ${index + 1}`);

      const img = document.createElement('img');
      img.src = src;
      img.alt = `thumb-${index + 1}`;

      btn.appendChild(img);

      btn.addEventListener('click', () => {
        if (mainImg) {
          mainImg.src = src;
        }
      });

      thumbs.appendChild(btn);
    });
  }

  function renderDetails(room) {
    const detailsList = document.querySelector('.details-card ul');

    if (!detailsList) return;

    detailsList.innerHTML = '';

    const capacityLi = document.createElement('li');
    capacityLi.textContent = `Макс. вместимость: ${room.max_capacity || room.capacity || 0}`;

    const addressLi = document.createElement('li');
    addressLi.textContent = `Адрес: ${room.address || '—'}`;

    const idLi = document.createElement('li');
    idLi.textContent = `ID помещения: ${room.id}`;

    detailsList.appendChild(capacityLi);
    detailsList.appendChild(addressLi);
    detailsList.appendChild(idLi);
  }

  function renderMap(room) {
    const mapWrap = document.getElementById('map');

    if (!mapWrap) return;

    if (mapInstance) {
      mapInstance.remove();
      mapInstance = null;
    }

    mapWrap.innerHTML = '';

    if (!room.lat || !room.lng || typeof L === 'undefined') {
      mapWrap.innerHTML = '<div style="padding:18px;color:var(--muted)">Координаты отсутствуют</div>';
      return;
    }

    try {
      mapInstance = L.map('map', {
        scrollWheelZoom: false
      }).setView([room.lat, room.lng], 15);

      mapInstance.attributionControl.setPrefix('');

      L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        maxZoom: 19,
        attribution: '&copy; OpenStreetMap contributors'
      }).addTo(mapInstance);

      L.marker([room.lat, room.lng]).addTo(mapInstance);
    } catch (err) {
      console.warn('Leaflet init failed', err);
      mapWrap.innerHTML = '<div style="padding:18px;color:var(--muted)">Не удалось загрузить карту</div>';
    }
  }

  function configureBookingButton() {
    const bookBtn = document.getElementById('bookBtn');

    if (!bookBtn) return;

    bookBtn.disabled = false;

    if (!isAuthenticated()) {
      showBookHint(true);
    } else {
      showBookHint(false);
    }

    bookBtn.addEventListener('click', (e) => {
      e.preventDefault();

      if (!isAuthenticated()) {
        navigate('/auth');
        return;
      }

      openBookingModal();
    });
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
    } else if (hint) {
      hint.remove();
    }
  }

  function bindBookingModalActions() {
    const bookingClose = document.getElementById('bookingClose');
    const bookingCancel = document.getElementById('bookingCancel');
    const bookingConfirmBtn = document.getElementById('bookingConfirm');
    const bookingModal = document.getElementById('bookingModal');

    if (bookingClose) {
      bookingClose.addEventListener('click', closeBookingModal);
    }

    if (bookingCancel) {
      bookingCancel.addEventListener('click', closeBookingModal);
    }

    if (bookingConfirmBtn) {
      bookingConfirmBtn.addEventListener('click', confirmBooking);
    }

    if (bookingModal) {
      bookingModal.addEventListener('click', (e) => {
        if (e.target === bookingModal) {
          closeBookingModal();
        }
      });
    }
  }

  async function openBookingModal() {
    if (!currentRoom?.id) {
      alert('Комната не загружена');
      return;
    }

    if (!isAuthenticated()) {
      navigate('/auth');
      return;
    }

    const bookingModal = document.getElementById('bookingModal');

    if (!bookingModal) return;

    resetBookingState();
    buildCalendarDays();
    renderCalendar();
    renderTimes([]);
    updateSummary();

    const bookingNotice = document.getElementById('bookingNotice');
    if (bookingNotice) {
      bookingNotice.textContent = 'Загружаем доступные слоты...';
    }

    bookingModal.classList.remove('hidden');
    bookingModal.setAttribute('aria-hidden', 'false');
    document.body.style.overflow = 'hidden';

    try {
      const availability = await window.Api.getRoomAvailability(currentRoom.id, 7);
      applyAvailability(availability);

      if (bookingNotice) {
        bookingNotice.textContent = 'Выберите дату и слоты';
      }
    } catch (err) {
      console.error(err);

      if (bookingNotice) {
        bookingNotice.textContent = err?.message || 'Не удалось загрузить доступные слоты';
      }
    }
  }

  function closeBookingModal() {
    const bookingModal = document.getElementById('bookingModal');

    if (!bookingModal) return;

    bookingModal.classList.add('hidden');
    bookingModal.setAttribute('aria-hidden', 'true');
    document.body.style.overflow = '';
  }

  function resetBookingState() {
    serverDatesMap = {};
    availableDatesSet = new Set();
    selectedDate = null;
    selectedSlots = [];
    calendarDays = [];
  }

  function applyAvailability(data) {
    const dates = Array.isArray(data?.dates)
      ? data.dates
      : Array.isArray(data)
        ? data
        : [];

    serverDatesMap = {};
    availableDatesSet = new Set();

    dates.forEach(d => {
      if (d && d.date && Array.isArray(d.available_times)) {
        serverDatesMap[d.date] = d.available_times;
        availableDatesSet.add(d.date);
      }
    });

    const first = calendarDays.find(cd => availableDatesSet.has(cd.ymd));

    if (first) {
      selectedDate = first.ymd;
    }

    renderCalendar();
    renderTimes(serverDatesMap[selectedDate] || []);
    updateSummary();
  }

  async function confirmBooking() {
    const bookingConfirm = document.getElementById('bookingConfirm');
    const bookingModal = document.getElementById('bookingModal');

    if (!selectedDate || selectedSlots.length === 0) {
      alert('Выберите дату и хотя бы один слот');
      return;
    }

    const orderedSlots = [...selectedSlots].sort((a, b) => parseTimeToMinutes(a) - parseTimeToMinutes(b));

    for (let i = 1; i < orderedSlots.length; i++) {
      if (parseTimeToMinutes(orderedSlots[i]) !== parseTimeToMinutes(orderedSlots[i - 1]) + 60) {
        alert('Слоты выбраны не подряд.');
        return;
      }
    }

    const payload = {
      room_id: Number(currentRoom.id),
      date: selectedDate,
      slots: orderedSlots
    };

    if (bookingConfirm) {
      bookingConfirm.disabled = true;
      bookingConfirm.textContent = 'Отправка...';
    }

    try {
      await window.Api.createBooking(payload);

      if (bookingModal) {
        const content = bookingModal.querySelector('.booking-modal__content');

        if (content) {
          content.innerHTML = `
            <h3>Бронирование успешно</h3>
            <div style="display:flex;gap:8px;margin-top:14px">
              <a href="/" class="link-as-btn primary">На главную</a>
              <a href="/me" class="link-as-btn">В кабинет</a>
            </div>
          `;
        }
      }
    } catch (err) {
      switch (err?.code) {
        case 'slot_already_booked':
          alert('Выбранный слот уже занят');
          break;

        case 'slots_must_be_consecutive':
          alert('Слоты должны идти подряд');
          break;

        case 'unauthorized':
          navigate('/auth');
          break;

        default:
          alert(err?.message || 'Ошибка бронирования');
      }
    } finally {
      if (bookingConfirm) {
        bookingConfirm.disabled = false;
        bookingConfirm.textContent = 'Подтвердить';
      }
    }
  }

  function buildCalendarDays() {
    calendarDays = [];

    const today = new Date();

    for (let i = 0; i < 7; i++) {
      const d = nextDay(today, i);

      calendarDays.push({
        ymd: formatYMD(d),
        dateObj: d
      });
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

      const dLabel = document.createElement('span');
      dLabel.className = 'd-date';
      dLabel.textContent = formatDayLabel(day.dateObj);

      const wLabel = document.createElement('span');
      wLabel.className = 'd-weekday';
      wLabel.textContent = weekdayRu(day.dateObj);

      div.appendChild(dLabel);
      div.appendChild(wLabel);

      if (availableDatesSet.has(day.ymd)) {
        div.classList.remove('disabled');
      } else {
        div.classList.add('disabled');
      }

      if (selectedDate === day.ymd) {
        div.classList.add('selected');
      } else {
        div.classList.remove('selected');
      }

      div.addEventListener('click', () => {
        if (!availableDatesSet.has(day.ymd)) return;
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

      if (selectedSlots.includes(t)) {
        btn.classList.add('selected');
      }

      btn.addEventListener('click', () => onSlotClick(t));

      bookingTimes.appendChild(btn);
    }
  }

  function onSlotClick(time) {
    const idx = selectedSlots.indexOf(time);

    if (idx !== -1) {
      if (selectedSlots.length === 1) {
        selectedSlots = [];
      } else {
        const sorted = [...selectedSlots].sort((a, b) => parseTimeToMinutes(a) - parseTimeToMinutes(b));

        if (time === sorted[0]) {
          sorted.shift();
          selectedSlots = sorted;
        } else if (time === sorted[sorted.length - 1]) {
          sorted.pop();
          selectedSlots = sorted;
        } else {
          selectedSlots = [time];
        }
      }

      renderTimes(serverDatesMap[selectedDate] || []);
      updateSummary();
      return;
    }

    if (selectedSlots.length === 0) {
      selectedSlots = [time];
    } else {
      const sorted = [...selectedSlots].sort((a, b) => parseTimeToMinutes(a) - parseTimeToMinutes(b));
      const first = sorted[0];
      const last = sorted[sorted.length - 1];

      if (parseTimeToMinutes(time) + 60 === parseTimeToMinutes(first)) {
        selectedSlots = [time, ...sorted];
      } else if (parseTimeToMinutes(time) === parseTimeToMinutes(last) + 60) {
        selectedSlots = [...sorted, time];
      } else {
        selectedSlots = [time];
      }
    }

    renderTimes(serverDatesMap[selectedDate] || []);
    updateSummary();
  }

  function updateSummary() {
    const selectionSummary = document.getElementById('selectionSummary');

    if (!selectionSummary) return;

    if (!selectedDate || selectedSlots.length === 0) {
      selectionSummary.textContent = 'Слоты: не выбраны';
      return;
    }

    const orderedSlots = [...selectedSlots].sort((a, b) => parseTimeToMinutes(a) - parseTimeToMinutes(b));

    selectionSummary.textContent = `Дата: ${selectedDate}, слоты: ${orderedSlots.join(', ')}`;
  }

  function renderRoomError(message) {
    const container = document.querySelector('.room-page') || document.querySelector('main');

    if (!container) {
      alert(message);
      return;
    }

    container.innerHTML = `
      <div class="center-message">
        <div>${escapeHtml(message)}</div>
        <div style="margin-top:12px">
          <a class="link-as-btn primary" href="/">На главную</a>
        </div>
      </div>
    `;
  }

  function setText(selector, value) {
    const node = document.querySelector(selector);

    if (node) {
      node.textContent = value;
    }
  }

  function navigate(url) {
    window.location.href = url;
  }

  function toNumberOrNull(value) {
    const n = Number(value);
    return Number.isFinite(n) ? n : null;
  }

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

  function escapeHtml(value) {
    return String(value)
      .replaceAll('&', '&amp;')
      .replaceAll('<', '&lt;')
      .replaceAll('>', '&gt;')
      .replaceAll('"', '&quot;')
      .replaceAll("'", '&#039;');
  }
})();