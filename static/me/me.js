let pendingCancelBookingId = null;

const uiHelpers = {
  reloadPage: () => window.location.reload()
};

function el(tag, props = {}) {
  const e = document.createElement(tag);
  Object.keys(props).forEach(k => {
    if (k === 'class') e.className = props[k];
    else if (k === 'text') e.textContent = props[k];
    else if (k === 'html') e.innerHTML = props[k];
    else e.setAttribute(k, props[k]);
  });
  return e;
}

function clearColumns() {
  const colInUse = document.getElementById('col-in_use');
  const colBooked = document.getElementById('col-booked');
  const colFinished = document.getElementById('col-finished');
  const colCanceled = document.getElementById('col-canceled');

  if (colInUse) colInUse.innerHTML = '';
  if (colBooked) colBooked.innerHTML = '';
  if (colFinished) colFinished.innerHTML = '';
  if (colCanceled) colCanceled.innerHTML = '';
}

function openConfirmCancel(bookingId) {
  pendingCancelBookingId = bookingId;

  const confirmModal = document.getElementById('confirmModal');
  const confirmText = document.getElementById('confirmText');

  if (confirmText) confirmText.textContent = 'Уверены, что хотите отменить бронь?';
  if (confirmModal) {
    confirmModal.classList.remove('hidden');
    confirmModal.setAttribute('aria-hidden', 'false');
  }
}

function closeConfirmCancel() {
  pendingCancelBookingId = null;

  const confirmModal = document.getElementById('confirmModal');
  if (confirmModal) {
    confirmModal.classList.add('hidden');
    confirmModal.setAttribute('aria-hidden', 'true');
  }
}

function createCard(b) {
  const btn = el('button', { class: 'me-card', type: 'button' });
  btn.dataset.bookingId = b.id;

  btn.addEventListener('click', (ev) => {
    if (ev.target.closest('.cancel-btn')) return;
    window.location.href = `/room/${encodeURIComponent(b.room_id)}`;
  });

  const img = el('img', { class: 'me-card__img', alt: b.title || '' });
  img.src = b.image_url || '/static/placeholders/room-placeholder.svg';

  const body = el('div', { class: 'me-card__body' });
  const title = el('div', { class: 'me-card__title', text: b.title || 'Без названия' });
  const date = el('div', { class: 'me-card__date', text: `${b.date}` });
  const time = el('div', { class: 'me-card__time', text: `${b.start_time} — ${b.end_time}` });
  const price = el('div', { class: 'me-card__price', text: `${b.total_price} ₽` });

  body.appendChild(title);
  body.appendChild(date);
  body.appendChild(time);
  body.appendChild(price);

  if (b.status === 'booked') {
    const cancelBtn = el('button', { class: 'cancel-btn', type: 'button', text: 'Отменить' });
    cancelBtn.addEventListener('click', (ev) => {
      ev.stopPropagation();
      openConfirmCancel(b.id);
    });
    body.appendChild(cancelBtn);
  }

  btn.appendChild(img);
  btn.appendChild(body);
  return btn;
}

function renderBookings(arr) {
  clearColumns();

  const emptyMessage = document.getElementById('emptyMessage');
  const colInUse = document.getElementById('col-in_use');
  const colBooked = document.getElementById('col-booked');
  const colFinished = document.getElementById('col-finished');
  const colCanceled = document.getElementById('col-canceled');

  if (!Array.isArray(arr) || arr.length === 0) {
    if (emptyMessage) emptyMessage.classList.remove('hidden');
    return;
  } else {
    if (emptyMessage) emptyMessage.classList.add('hidden');
  }

  for (const b of arr) {
    const card = createCard(b);
    switch (b.status) {
      case 'in_use':
        if (colInUse) colInUse.appendChild(card);
        break;
      case 'booked':
        if (colBooked) colBooked.appendChild(card);
        break;
      case 'finished':
        if (colFinished) colFinished.appendChild(card);
        break;
      case 'canceled':
        if (colCanceled) colCanceled.appendChild(card);
        break;
      default:
        if (colFinished) colFinished.appendChild(card);
    }
  }
}

async function cancelBooking(bookingId) {
  if (!bookingId) return;
  try {
    const res = await fetch('/api/booking/stop', {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ booking_id: bookingId })
    });

if (res.ok) {
    uiHelpers.reloadPage(); 
    return;
  }

    switch (res.status) {
      case 400:
        alert('Некорректные данные');
        break;
      case 403:
        alert('Попытка отменить чужую бронь');
        break;
      case 404:
        alert('Бронь не найдена');
        break;
      case 409:
        alert('Бронь уже используется или завершена — обновляем страницу');
        uiHelpers.reloadPage();
        break;
      case 500:
      default:
        alert('Ошибка доступа к серверу. Повторите попытку.');
    }
  } catch (err) {
    console.error(err);
    alert('Ошибка соединения');
  }
}

function getWsUrl() {
  const scheme = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${scheme}//${window.location.host}/ws/me`;
}

function connectWs() {
  let ws = null;

  try {
    ws = new WebSocket(getWsUrl());
  } catch (err) {
    console.error('Не удалось создать WebSocket:', err);
    return;
  }

  ws.addEventListener('open', () => {
  });

  ws.addEventListener('message', (ev) => {
    try {
      const data = JSON.parse(ev.data);
      if (!Array.isArray(data)) {
        console.warn('Ожидался массив бронирований от /ws/me, получено:', data);
        return;
      }
      renderBookings(data);
    } catch (err) {
      console.warn('Ошибка разбора сообщения /ws/me:', err);
    }
  });

  ws.addEventListener('close', () => {
  });

  ws.addEventListener('error', (e) => {
    console.error('ws /ws/me error', e);
  });
}

document.addEventListener('DOMContentLoaded', () => {
  const serverUser = window.__USER__ || {};
  const isAuthServer = !!serverUser.auth;

  const logoutBtn = document.getElementById('logoutBtn');
  const bottomBar = document.getElementById('meBottomBar');
  const confirmModal = document.getElementById('confirmModal');
  const confirmYes = document.getElementById('confirmYes');
  const confirmNo = document.getElementById('confirmNo');

  if (!isAuthServer) {
    if (bottomBar) bottomBar.remove();
    if (logoutBtn) logoutBtn.style.display = 'none';
    return;
  }

  connectWs();

  if (confirmYes) {
    confirmYes.addEventListener('click', () => {
      if (pendingCancelBookingId) cancelBooking(pendingCancelBookingId);
      closeConfirmCancel();
    });
  }

  if (confirmNo) {
    confirmNo.addEventListener('click', () => {
      closeConfirmCancel();
    });
  }

  if (confirmModal) {
    confirmModal.addEventListener('click', (e) => {
      if (e.target === confirmModal) closeConfirmCancel();
    });
  }

  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') closeConfirmCancel();
  });

  if (logoutBtn) {
    logoutBtn.addEventListener('click', async () => {
      try {
        const res = await fetch('/api/logout', {
          method: 'POST',
          credentials: 'same-origin'
        });
        if (res.ok) {
          window.location.href = '/';
          return;
        }
        switch (res.status) {
          case 400:
            alert('Сессия отсутствует или истекла');
            break;
          case 500:
          default:
            alert('Ошибка доступа к серверу. Повторите попытку.');
            break;
        }
      } catch (err) {
        console.error(err);
        alert('Ошибка соединения');
      }
    });
  }

  if (bottomBar && !logoutBtn) {
    const center = bottomBar.querySelector('.me-bottombar__center') || bottomBar.querySelector('.me-bottombar__inner');
    if (center) {
      const btn = el('button', { id: 'logoutBtn', class: 'btn', text: 'Выйти', type: 'button' });
      btn.addEventListener('click', async () => {
        try {
          const r = await fetch('/api/logout', { method: 'POST', credentials: 'same-origin' });
          if (r.ok) window.location.href = '/';
          else alert('Ошибка выхода');
        } catch (err) {
          console.error(err);
          alert('Ошибка соединения');
        }
      });
      center.appendChild(btn);
    }
  }
});

if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    renderBookings,
    createCard,
    clearColumns,
    openConfirmCancel,
    closeConfirmCancel,
    cancelBooking,
    connectWs,
    getWsUrl,
    uiHelpers
  };
}