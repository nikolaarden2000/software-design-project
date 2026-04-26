(function () {
  'use strict';

  let pendingCancelBookingId = null;
  let currentBookings = [];

  document.addEventListener('DOMContentLoaded', initMePage);

  function el(tag, props = {}) {
    const e = document.createElement(tag);

    Object.keys(props).forEach(k => {
      if (k === 'class') {
        e.className = props[k];
      } else if (k === 'text') {
        e.textContent = props[k];
      } else if (k === 'html') {
        e.innerHTML = props[k];
      } else {
        e.setAttribute(k, props[k]);
      }
    });

    return e;
  }

  async function initMePage() {
    bindModalEvents();
    bindLogout();

    try {
      const me = await window.Api.getMe();

      if (!me?.authenticated) {
        window.location.href = '/auth';
        return;
      }

      await loadBookings();
    } catch (err) {
      console.error(err);
      alert('Ошибка загрузки личного кабинета');
    }
  }

  async function loadBookings() {
    try {
      const data = await window.Api.getMyBookings();

      const items = Array.isArray(data?.items)
        ? data.items
        : Array.isArray(data)
          ? data
          : [];

      currentBookings = items;
      renderBookings(currentBookings);
    } catch (err) {
      console.error('Ошибка загрузки бронирований:', err);

      if (err?.code === 'unauthorized') {
        window.location.href = '/auth';
        return;
      }

      alert(err?.message || 'Не удалось загрузить бронирования');
    }
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
    }

    if (emptyMessage) {
      emptyMessage.classList.add('hidden');
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

  function createCard(b) {
    const btn = el('button', {
      class: 'me-card',
      type: 'button'
    });

    btn.dataset.bookingId = b.id;

    btn.addEventListener('click', (ev) => {
      if (ev.target.closest('.cancel-btn')) return;
      window.location.href = `/room/${encodeURIComponent(b.room_id)}`;
    });

    const img = el('img', {
      class: 'me-card__img',
      alt: b.title || ''
    });

    img.src = b.image_url || '/shared/placeholders/room-placeholder.svg';

    const body = el('div', {
      class: 'me-card__body'
    });

    const title = el('div', {
      class: 'me-card__title',
      text: b.title || 'Без названия'
    });

    const date = el('div', {
      class: 'me-card__date',
      text: `${b.date || ''}`
    });

    const time = el('div', {
      class: 'me-card__time',
      text: `${b.start_time || ''} — ${b.end_time || ''}`
    });

    const price = el('div', {
      class: 'me-card__price',
      text: `${b.total_price || 0} ₽`
    });

    body.appendChild(title);
    body.appendChild(date);
    body.appendChild(time);
    body.appendChild(price);

    if (b.status === 'booked') {
      const cancelBtn = el('button', {
        class: 'cancel-btn',
        type: 'button',
        text: 'Отменить'
      });

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

  function openConfirmCancel(bookingId) {
    pendingCancelBookingId = bookingId;

    const confirmModal = document.getElementById('confirmModal');
    const confirmText = document.getElementById('confirmText');

    if (confirmText) {
      confirmText.textContent = 'Уверены, что хотите отменить бронь?';
    }

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

  async function cancelBooking(bookingId) {
    if (!bookingId) return;

    try {
      const result = await window.Api.cancelBooking(bookingId);

      currentBookings = currentBookings.map(b => {
        if (String(b.id) !== String(bookingId)) {
          return b;
        }

        return {
          ...b,
          status: result?.status || 'canceled'
        };
      });

      renderBookings(currentBookings);

      await loadBookings();
    } catch (err) {
      console.error(err);

      if (err?.code === 'unauthorized') {
        alert('Необходимо авторизоваться');
        window.location.href = '/auth';
        return;
      }

      if (err?.code === 'cannot_cancel_booking') {
        alert(err.message || 'Бронь уже используется или завершена');
        await loadBookings();
        return;
      }

      alert(err?.message || 'Ошибка отмены бронирования');
    }
  }

  function bindModalEvents() {
    const confirmModal = document.getElementById('confirmModal');
    const confirmYes = document.getElementById('confirmYes');
    const confirmNo = document.getElementById('confirmNo');

    if (confirmYes) {
      confirmYes.addEventListener('click', async () => {
        const bookingId = pendingCancelBookingId;
        closeConfirmCancel();

        if (bookingId) {
          await cancelBooking(bookingId);
        }
      });
    }

    if (confirmNo) {
      confirmNo.addEventListener('click', closeConfirmCancel);
    }

    if (confirmModal) {
      confirmModal.addEventListener('click', (e) => {
        if (e.target === confirmModal) {
          closeConfirmCancel();
        }
      });
    }

    document.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        closeConfirmCancel();
      }
    });
  }

  function bindLogout() {
    const logoutBtn = document.getElementById('logoutBtn');

    if (!logoutBtn) return;

    logoutBtn.addEventListener('click', async () => {
      try {
        await window.Api.logoutUser();
        window.location.href = '/';
      } catch (err) {
        console.error(err);
        alert(err?.message || 'Ошибка выхода');
      }
    });
  }
})();