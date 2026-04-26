(function () {
  'use strict';

  let locations = [];
  let rooms = [];
  let bookings = [];

  document.addEventListener('DOMContentLoaded', initAdminPage);

  async function initAdminPage() {
    bindLogout();
    bindForms();
    bindFilters();

    try {
      const me = await window.Api.getMe();

      if (!me?.authenticated) {
        window.location.href = '/auth';
        return;
      }

      if (me.user?.role !== 'admin' && me.user?.role !== 'superuser') {
        showAccessDenied();
        return;
      }

      showContent();
      await loadAllData();
    } catch (err) {
      console.error(err);
      alert(err?.message || 'Ошибка загрузки панели администратора');
    }
  }

  function showAccessDenied() {
    document.getElementById('accessDenied')?.classList.remove('hidden');
    document.getElementById('adminContent')?.classList.add('hidden');
  }

  function showContent() {
    document.getElementById('accessDenied')?.classList.add('hidden');
    document.getElementById('adminContent')?.classList.remove('hidden');
  }

  async function loadAllData() {
    await loadLocations();

    fillLocationSelects();

    await Promise.all([
      loadRooms(),
      loadBookings()
    ]);
  }

  async function loadLocations() {
    const data = await window.Api.getAdminLocations();
    locations = data.items || [];
    renderLocations();
  }

  async function loadRooms() {
    const params = {};

    const locationId = document.getElementById('roomsLocationFilter')?.value;
    const status = document.getElementById('roomsStatusFilter')?.value;

    if (locationId) {
      params.location_id = locationId;
    }

    if (status) {
      params.status = status;
    }

    const data = await window.Api.getAdminRooms(params);
    rooms = data.items || [];
    renderRooms();
  }

  async function loadBookings() {
    const params = {};

    const locationId = document.getElementById('bookingsLocationFilter')?.value;
    const status = document.getElementById('bookingsStatusFilter')?.value;

    if (locationId) {
      params.location_id = locationId;
    }

    if (status) {
      params.status = status;
    }

    const data = await window.Api.getAdminBookings(params);
    bookings = data.items || [];
    renderBookings();
  }

  function renderLocations() {
    const root = document.getElementById('locationsList');
    if (!root) return;

    if (locations.length === 0) {
      root.innerHTML = '<div class="item">У вас пока нет назначенных локаций</div>';
      return;
    }

    root.innerHTML = locations.map(location => `
      <div class="item">
        <div class="item-title">${escapeHtml(location.company_name)}</div>
        <div class="item-meta">${escapeHtml(location.city)}, ${escapeHtml(location.address)}</div>
        <div class="item-meta">Комнат: ${location.rooms_count ?? 0}</div>
        <div class="item-meta">Таймзона: ${escapeHtml(location.timezone || '')}</div>
      </div>
    `).join('');
  }

  function renderRooms() {
    const root = document.getElementById('roomsList');
    if (!root) return;

    if (rooms.length === 0) {
      root.innerHTML = '<div class="item">Помещений пока нет</div>';
      return;
    }

    root.innerHTML = rooms.map(room => {
      const statusLabel = getRoomStatusLabel(room.status);
      const canSubmit = room.status === 'draft' || room.status === 'rejected';

      return `
        <article class="room-card" data-room-id="${room.id}">
          <h3>${escapeHtml(room.title)}</h3>

          <div class="item-meta">
            Локация ID: ${room.location_id}
          </div>

          <div class="card-meta">
            <span class="badge">${room.price} ₽/ч</span>
            <span class="badge">до ${room.capacity} чел.</span>
            <span class="badge badge-${escapeHtml(room.status)}">${statusLabel}</span>
          </div>

          ${room.rejection_reason ? `
            <div class="item-meta">
              Причина отклонения: ${escapeHtml(room.rejection_reason)}
            </div>
          ` : ''}

          <div class="card-actions">
            <button 
              class="btn success"
              data-action="submit-room"
              data-room-id="${room.id}"
              ${canSubmit ? '' : 'disabled'}
            >
              Отправить на модерацию
            </button>
          </div>
        </article>
      `;
    }).join('');

    root.querySelectorAll('[data-action="submit-room"]').forEach(button => {
      button.addEventListener('click', () => submitRoom(button.dataset.roomId));
    });
  }

  function renderBookings() {
    const root = document.getElementById('bookingsList');
    if (!root) return;

    if (bookings.length === 0) {
      root.innerHTML = '<div class="item">Бронирований пока нет</div>';
      return;
    }

    root.innerHTML = bookings.map(booking => {
      const canCancel = booking.status === 'booked';

      return `
        <article class="booking-card" data-booking-id="${booking.id}">
          <h3>${escapeHtml(booking.room_title)}</h3>

          <div class="item-meta">
            ${escapeHtml(booking.location_address || '')}
          </div>

          <div class="card-meta">
            <span class="badge">${escapeHtml(booking.date)}</span>
            <span class="badge">${escapeHtml(booking.start_time)} — ${escapeHtml(booking.end_time)}</span>
            <span class="badge">${booking.total_price} ₽</span>
            <span class="badge">${getBookingStatusLabel(booking.status)}</span>
          </div>

          <div class="item-meta">
            Пользователь: ${escapeHtml(booking.user_username || '')}
            (${escapeHtml(booking.user_email || '')})
          </div>

          <div class="card-actions">
            <button
              class="btn danger"
              data-action="cancel-booking"
              data-booking-id="${booking.id}"
              ${canCancel ? '' : 'disabled'}
            >
              Отменить бронь
            </button>
          </div>
        </article>
      `;
    }).join('');

    root.querySelectorAll('[data-action="cancel-booking"]').forEach(button => {
      button.addEventListener('click', () => cancelBooking(button.dataset.bookingId));
    });
  }

  function fillLocationSelects() {
    const selects = [
      document.getElementById('roomLocation'),
      document.getElementById('roomsLocationFilter'),
      document.getElementById('bookingsLocationFilter')
    ];

    selects.forEach(select => {
      if (!select) return;

      const firstOption = select.querySelector('option')?.cloneNode(true);
      select.innerHTML = '';

      if (firstOption) {
        select.appendChild(firstOption);
      }

      locations.forEach(location => {
        const option = document.createElement('option');
        option.value = location.id;
        option.textContent = `${location.company_name} — ${location.city}, ${location.address}`;
        select.appendChild(option);
      });
    });
  }

  function bindForms() {
    const roomForm = document.getElementById('roomForm');

    if (!roomForm) return;

    roomForm.addEventListener('submit', async (event) => {
      event.preventDefault();

      const imageValue = document.getElementById('roomImages').value.trim();

      const payload = {
        location_id: Number(document.getElementById('roomLocation').value),
        title: document.getElementById('roomTitle').value.trim(),
        description: document.getElementById('roomDescription').value.trim(),
        price: Number(document.getElementById('roomPrice').value),
        capacity: Number(document.getElementById('roomCapacity').value),
        available_from: document.getElementById('roomAvailableFrom').value,
        available_to: document.getElementById('roomAvailableTo').value,
        images: imageValue
          ? imageValue.split(',').map(item => item.trim()).filter(Boolean)
          : ['/shared/placeholders/room-placeholder.svg']
      };

      if (!payload.location_id) {
        alert('Выберите локацию');
        return;
      }

      if (!payload.title || !payload.description) {
        alert('Введите название и описание помещения');
        return;
      }

      if (payload.price <= 0 || payload.capacity <= 0) {
        alert('Цена и вместимость должны быть больше нуля');
        return;
      }

      try {
        await window.Api.createAdminRoom(payload);

        roomForm.reset();
        document.getElementById('roomAvailableFrom').value = '09:00';
        document.getElementById('roomAvailableTo').value = '21:00';

        await Promise.all([
          loadRooms(),
          loadLocations()
        ]);

        fillLocationSelects();

        alert('Помещение создано как черновик');
      } catch (err) {
        console.error(err);
        alert(err?.message || 'Ошибка создания помещения');
      }
    });
  }

  function bindFilters() {
    const reloadRoomsBtn = document.getElementById('reloadRoomsBtn');
    const reloadBookingsBtn = document.getElementById('reloadBookingsBtn');

    const roomsLocationFilter = document.getElementById('roomsLocationFilter');
    const roomsStatusFilter = document.getElementById('roomsStatusFilter');

    const bookingsLocationFilter = document.getElementById('bookingsLocationFilter');
    const bookingsStatusFilter = document.getElementById('bookingsStatusFilter');

    if (reloadRoomsBtn) {
      reloadRoomsBtn.addEventListener('click', loadRooms);
    }

    if (reloadBookingsBtn) {
      reloadBookingsBtn.addEventListener('click', loadBookings);
    }

    if (roomsLocationFilter) {
      roomsLocationFilter.addEventListener('change', loadRooms);
    }

    if (roomsStatusFilter) {
      roomsStatusFilter.addEventListener('change', loadRooms);
    }

    if (bookingsLocationFilter) {
      bookingsLocationFilter.addEventListener('change', loadBookings);
    }

    if (bookingsStatusFilter) {
      bookingsStatusFilter.addEventListener('change', loadBookings);
    }
  }

  async function submitRoom(roomId) {
    try {
      await window.Api.submitRoomForModeration(roomId);
      await loadRooms();
      alert('Помещение отправлено на модерацию');
    } catch (err) {
      console.error(err);
      alert(err?.message || 'Ошибка отправки помещения на модерацию');
    }
  }

  async function cancelBooking(bookingId) {
    const reason = prompt('Укажите причину отмены бронирования');

    if (reason === null) {
      return;
    }

    try {
      await window.Api.cancelAdminBooking(bookingId, reason.trim() || 'Отменено администратором');
      await loadBookings();
      alert('Бронирование отменено');
    } catch (err) {
      console.error(err);
      alert(err?.message || 'Ошибка отмены бронирования');
    }
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

  function getRoomStatusLabel(status) {
    const map = {
      draft: 'Черновик',
      pending: 'На модерации',
      published: 'Опубликовано',
      rejected: 'Отклонено',
      archived: 'Архивировано'
    };

    return map[status] || status;
  }

  function getBookingStatusLabel(status) {
    const map = {
      booked: 'Забронировано',
      in_use: 'Используется',
      finished: 'Завершено',
      canceled: 'Отменено'
    };

    return map[status] || status;
  }

  function escapeHtml(value) {
    return String(value ?? '')
      .replaceAll('&', '&amp;')
      .replaceAll('<', '&lt;')
      .replaceAll('>', '&gt;')
      .replaceAll('"', '&quot;')
      .replaceAll("'", '&#039;');
  }
})();