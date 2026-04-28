(function () {
  'use strict';

  const USE_MOCKS = false;

  // Для проверки страниц:
  // 'user'       -> обычный пользователь
  // 'admin'      -> /admin
  // 'superuser'  -> /superuser и /admin
  const MOCK_ROLE = 'admin';

  const MOCK_STORAGE_KEY = 'booking_mock_state_v4';

  async function apiRequest(path, options = {}) {
    if (USE_MOCKS) {
      return mockRequest(path, options);
    }

    const headers = options.body
      ? { 'Content-Type': 'application/json', ...(options.headers || {}) }
      : { ...(options.headers || {}) };

    const res = await fetch(path, {
      credentials: 'include',
      headers,
      ...options
    });

    if (res.status === 204) {
      return null;
    }

    const body = await res.json().catch(() => null);

    if (!res.ok) {
      throw body?.error || {
        code: 'unknown_error',
        message: 'Неизвестная ошибка'
      };
    }

    return body?.data ?? body;
  }

  async function mockRequest(path, options = {}) {
    await new Promise(resolve => setTimeout(resolve, 150));

    const method = (options.method || 'GET').toUpperCase();
    const url = new URL(path, window.location.origin);
    const pathname = url.pathname;
    const state = getMockState();

    if (pathname === '/api/me') {
      return {
        authenticated: true,
        user: getMockUser()
      };
    }

    if (pathname === '/api/login' && method === 'POST') {
      return {
        user: getMockUser()
      };
    }

    if (pathname === '/api/register' && method === 'POST') {
      return {
        id: 1000,
        username: 'new-user',
        email: 'new-user@mail.com',
        role: 'user'
      };
    }

    if (pathname === '/api/logout' && method === 'POST') {
      return null;
    }

    if (pathname === '/api/cities') {
      return {
        items: ['Москва', 'Санкт-Петербург', 'Казань', 'Екатеринбург']
      };
    }

    // =========================================================
    // Public user API
    // =========================================================

    if (pathname === '/api/rooms/filters') {
      const city = url.searchParams.get('city') || 'Москва';

      const publishedRooms = state.rooms.filter(room => {
        const location = getLocationById(state, room.location_id);
        return room.status === 'published' && location?.city === city;
      });

      const companies = [
        ...new Set(
          publishedRooms
            .map(room => getLocationById(state, room.location_id)?.company_name)
            .filter(Boolean)
        )
      ];

      return {
        city,
        companies,
        price: {
          min: 500,
          max: 5000
        },
        capacity: {
          min: 2,
          max: 30
        }
      };
    }

    if (pathname === '/api/rooms' && method === 'GET') {
      const city = url.searchParams.get('city') || '';
      const company = url.searchParams.get('company') || '';
      const maxPrice = Number(url.searchParams.get('max_price') || 0);
      const minCapacity = Number(url.searchParams.get('min_capacity') || 0);

      let rooms = state.rooms.filter(room => room.status === 'published');

      if (city) {
        rooms = rooms.filter(room => getLocationById(state, room.location_id)?.city === city);
      }

      if (company) {
        rooms = rooms.filter(room => getLocationById(state, room.location_id)?.company_name === company);
      }

      if (maxPrice > 0) {
        rooms = rooms.filter(room => room.price <= maxPrice);
      }

      if (minCapacity > 0) {
        rooms = rooms.filter(room => room.capacity >= minCapacity);
      }

      return {
        items: rooms.map(room => {
          const location = getLocationById(state, room.location_id);

          return {
            id: room.id,
            title: room.title,
            company: location?.company_name || '',
            address: location?.address || '',
            capacity: room.capacity,
            image: room.images?.[0] || '/shared/placeholders/room-placeholder.svg',
            price: room.price
          };
        }),
        pagination: {
          limit: Number(url.searchParams.get('limit') || 100),
          next_after_id: null,
          has_more: false
        }
      };
    }

    const publicRoomMatch = pathname.match(/^\/api\/rooms\/(\d+)$/);
    if (publicRoomMatch && method === 'GET') {
      const roomId = Number(publicRoomMatch[1]);
      const room = state.rooms.find(item => item.id === roomId && item.status === 'published');

      if (!room) {
        throwError('room_not_found', 'Комната не найдена');
      }

      const location = getLocationById(state, room.location_id);

      return {
        id: room.id,
        title: room.title,
        company: location?.company_name || '',
        address: location?.address || '',
        images: room.images,
        price: room.price,
        currency: 'RUB',
        capacity: room.capacity,
        max_capacity: room.capacity,
        available_from: room.available_from,
        available_to: room.available_to,
        description: room.description,
        lat: location?.lat,
        lng: location?.lng
      };
    }

    const availabilityMatch = pathname.match(/^\/api\/rooms\/(\d+)\/availability$/);
    if (availabilityMatch && method === 'GET') {
      return {
        room_id: Number(availabilityMatch[1]),
        dates: [
          {
            date: getDatePlusDays(1),
            available_times: ['10:00', '11:00', '12:00', '15:00']
          },
          {
            date: getDatePlusDays(2),
            available_times: ['09:00', '13:00', '14:00', '18:00']
          },
          {
            date: getDatePlusDays(3),
            available_times: ['10:00', '16:00', '17:00']
          }
        ]
      };
    }

    if (pathname === '/api/bookings' && method === 'POST') {
      const payload = parseJson(options.body);
      const room = getRoomById(state, Number(payload.room_id));
      const location = room ? getLocationById(state, room.location_id) : null;

      if (!room || room.status !== 'published') {
        throwError('room_not_found', 'Комната не найдена');
      }

      if (room.archive?.booking_disabled) {
        throwError('booking_disabled', 'Для помещения отключены новые бронирования');
      }

      const slots = Array.isArray(payload.slots) ? payload.slots : [];
      const startTime = slots[0] || '10:00';
      const endTime = slots.length > 0 ? addOneHour(slots[slots.length - 1]) : '11:00';

      const booking = {
        id: state.nextBookingId++,
        room_id: room.id,
        room_title: room.title,
        location_id: room.location_id,
        location_address: location?.address || '',
        user_id: 100,
        user_email: 'user@mail.com',
        user_username: 'Тестовый пользователь',
        date: payload.date,
        start_time: startTime,
        end_time: endTime,
        total_price: room.price * Math.max(slots.length, 1),
        status: 'booked'
      };

      state.bookings.push(booking);
      saveMockState(state);

      return {
        id: booking.id,
        room_id: booking.room_id,
        date: booking.date,
        start_time: booking.start_time,
        end_time: booking.end_time,
        status: booking.status
      };
    }

    if (pathname === '/api/me/bookings' && method === 'GET') {
      return {
        items: state.bookings.map(booking => ({
          id: booking.id,
          room_id: booking.room_id,
          image_url: '/shared/placeholders/room-placeholder.svg',
          title: booking.room_title,
          date: booking.date,
          start_time: booking.start_time,
          end_time: booking.end_time,
          total_price: booking.total_price,
          status: booking.status
        }))
      };
    }

    const userCancelMatch = pathname.match(/^\/api\/bookings\/(\d+)\/cancel$/);
    if (userCancelMatch && method === 'POST') {
      const booking = getBookingById(state, Number(userCancelMatch[1]));

      if (!booking) {
        throwError('booking_not_found', 'Бронь не найдена');
      }

      booking.status = 'canceled';
      saveMockState(state);

      return {
        id: booking.id,
        status: booking.status
      };
    }

    // =========================================================
    // Admin API
    // =========================================================

    if (pathname === '/api/admin/locations' && method === 'GET') {
      return {
        items: getVisibleAdminLocations(state).map(location => ({
          ...location,
          rooms_count: state.rooms.filter(room => room.location_id === location.id).length
        }))
      };
    }

    if (pathname === '/api/admin/rooms' && method === 'GET') {
      const locationId = Number(url.searchParams.get('location_id') || 0);
      const status = url.searchParams.get('status');
      const visibleLocationIds = getVisibleAdminLocations(state).map(location => location.id);

      let rooms = state.rooms.filter(room => visibleLocationIds.includes(room.location_id));

      if (locationId) {
        rooms = rooms.filter(room => room.location_id === locationId);
      }

      if (status) {
        rooms = rooms.filter(room => room.status === status);
      }

      return {
        items: rooms.map(toAdminRoomListItem)
      };
    }

    if (pathname === '/api/admin/rooms' && method === 'POST') {
      const payload = parseJson(options.body);

      const room = {
        id: state.nextRoomId++,
        location_id: Number(payload.location_id),
        title: payload.title,
        description: payload.description,
        price: Number(payload.price),
        capacity: Number(payload.capacity),
        available_from: payload.available_from,
        available_to: payload.available_to,
        images: normalizeImages(payload.images),
        status: 'draft',
        rejection_reason: null,
        created_at: new Date().toISOString(),
        archive: {
          can_archive_now: true,
          has_active_or_future_bookings: false,
          booking_disabled: false,
          scheduled_for: null
        }
      };

      state.rooms.push(room);
      saveMockState(state);

      return toAdminRoomListItem(room);
    }

    const adminRoomMatch = pathname.match(/^\/api\/admin\/rooms\/(\d+)$/);
    if (adminRoomMatch && method === 'GET') {
      const room = getRoomById(state, Number(adminRoomMatch[1]));

      if (!room) {
        throwError('room_not_found', 'Помещение не найдено');
      }

      refreshRoomArchiveInfo(state, room);
      saveMockState(state);

      return {
        ...room,
        archive: room.archive
      };
    }

    if (adminRoomMatch && method === 'PATCH') {
      const room = getRoomById(state, Number(adminRoomMatch[1]));

      if (!room) {
        throwError('room_not_found', 'Помещение не найдено');
      }

      if (room.status !== 'draft' && room.status !== 'rejected') {
        throwError('cannot_edit_room', 'Можно редактировать только черновик или отклонённое помещение');
      }

      const payload = parseJson(options.body);

      room.location_id = Number(payload.location_id);
      room.title = payload.title;
      room.description = payload.description;
      room.price = Number(payload.price);
      room.capacity = Number(payload.capacity);
      room.available_from = payload.available_from;
      room.available_to = payload.available_to;
      room.images = normalizeImages(payload.images);

      if (room.status === 'rejected') {
        room.status = 'draft';
        room.rejection_reason = null;
      }

      saveMockState(state);

      return {
        id: room.id,
        status: room.status
      };
    }

    const submitRoomMatch = pathname.match(/^\/api\/admin\/rooms\/(\d+)\/submit$/);
    if (submitRoomMatch && method === 'POST') {
      const room = getRoomById(state, Number(submitRoomMatch[1]));

      if (!room) {
        throwError('room_not_found', 'Помещение не найдено');
      }

      if (room.status !== 'draft' && room.status !== 'rejected') {
        throwError('cannot_submit_room', 'Можно отправить только черновик или отклонённое помещение');
      }

      room.status = 'pending';
      saveMockState(state);

      return {
        id: room.id,
        status: 'pending'
      };
    }

    const archiveRoomMatch = pathname.match(/^\/api\/admin\/rooms\/(\d+)\/archive$/);
    if (archiveRoomMatch && method === 'POST') {
      const room = getRoomById(state, Number(archiveRoomMatch[1]));

      if (!room) {
        throwError('room_not_found', 'Помещение не найдено');
      }

      const payload = parseJson(options.body);
      const mode = payload.mode;

      refreshRoomArchiveInfo(state, room);

      if (mode === 'immediate') {
        if (room.archive.has_active_or_future_bookings) {
          throwError(
            'room_has_active_bookings',
            'Нельзя архивировать помещение, пока есть действующие или будущие бронирования'
          );
        }

        room.status = 'archived';
        room.archive.can_archive_now = false;
        room.archive.booking_disabled = true;
        room.archive.scheduled_for = null;

        saveMockState(state);

        return {
          id: room.id,
          status: 'archived'
        };
      }

      if (mode === 'scheduled') {
        room.archive.booking_disabled = true;
        room.archive.scheduled_for = getDatePlusDaysIso(14);

        saveMockState(state);

        return {
          id: room.id,
          status: room.status,
          booking_disabled: true,
          archive_scheduled_for: room.archive.scheduled_for
        };
      }

      throwError('invalid_request', 'Некорректный режим архивирования');
    }

    if (pathname === '/api/admin/bookings' && method === 'GET') {
      const roomId = Number(url.searchParams.get('room_id') || 0);
      const locationId = Number(url.searchParams.get('location_id') || 0);
      const status = url.searchParams.get('status');

      let bookings = [...state.bookings];

      if (roomId) {
        bookings = bookings.filter(booking => booking.room_id === roomId);
      }

      if (locationId) {
        bookings = bookings.filter(booking => booking.location_id === locationId);
      }

      if (status) {
        bookings = bookings.filter(booking => booking.status === status);
      }

      return {
        items: bookings
      };
    }

    const adminCancelBookingMatch = pathname.match(/^\/api\/admin\/bookings\/(\d+)\/cancel$/);
    if (adminCancelBookingMatch && method === 'POST') {
      const booking = getBookingById(state, Number(adminCancelBookingMatch[1]));

      if (!booking) {
        throwError('booking_not_found', 'Бронь не найдена');
      }

      if (booking.status !== 'booked') {
        throwError('cannot_cancel_booking', 'Можно отменить только активную будущую бронь');
      }

      booking.status = 'canceled';
      saveMockState(state);

      return {
        id: booking.id,
        status: 'canceled'
      };
    }

    // =========================================================
    // Superuser API
    // =========================================================

    if (pathname === '/api/superuser/companies') {
      if (method === 'POST') {
        const payload = parseJson(options.body);

        const company = {
          id: state.nextCompanyId++,
          name: payload.name,
          description: payload.description || '',
          locations_count: 0
        };

        state.companies.push(company);
        saveMockState(state);

        return company;
      }

      return {
        items: state.companies.map(company => ({
          ...company,
          locations_count: state.locations.filter(location => location.company_id === company.id).length
        }))
      };
    }

    if (pathname === '/api/superuser/locations') {
      if (method === 'POST') {
        const payload = parseJson(options.body);
        const company = state.companies.find(item => item.id === Number(payload.company_id));

        if (!company) {
          throwError('company_not_found', 'Компания не найдена');
        }

        const location = {
          id: state.nextLocationId++,
          company_id: Number(payload.company_id),
          company_name: company.name,
          city: payload.city,
          address: payload.address,
          lat: Number(payload.lat),
          lng: Number(payload.lng),
          timezone: payload.timezone || 'Europe/Moscow',
          rooms_count: 0
        };

        state.locations.push(location);
        saveMockState(state);

        return location;
      }

      return {
        items: state.locations.map(location => ({
          ...location,
          rooms_count: state.rooms.filter(room => room.location_id === location.id).length
        }))
      };
    }

    if (pathname === '/api/superuser/admins') {
      if (method === 'POST') {
        const payload = parseJson(options.body);

        const admin = {
          id: state.nextAdminId++,
          username: payload.username,
          email: payload.email,
          role: 'admin',
          locations: []
        };

        state.admins.push(admin);
        saveMockState(state);

        return admin;
      }

      return {
        items: state.admins
      };
    }

    const assignAdminMatch = pathname.match(/^\/api\/superuser\/admins\/(\d+)\/locations$/);
    if (assignAdminMatch && method === 'POST') {
      const adminId = Number(assignAdminMatch[1]);
      const payload = parseJson(options.body);
      const locationId = Number(payload.location_id);

      const admin = state.admins.find(item => item.id === adminId);
      const location = state.locations.find(item => item.id === locationId);

      if (!admin) {
        throwError('admin_not_found', 'Администратор не найден');
      }

      if (!location) {
        throwError('location_not_found', 'Локация не найдена');
      }

      const alreadyAssigned = admin.locations.some(item => item.id === locationId);

      if (!alreadyAssigned) {
        admin.locations.push({
          id: location.id,
          address: location.address,
          company_name: location.company_name
        });
      }

      saveMockState(state);

      return {
        admin_id: adminId,
        location_id: locationId
      };
    }

    const deleteAssignMatch = pathname.match(/^\/api\/superuser\/admins\/(\d+)\/locations\/(\d+)$/);
    if (deleteAssignMatch && method === 'DELETE') {
      const adminId = Number(deleteAssignMatch[1]);
      const locationId = Number(deleteAssignMatch[2]);

      const admin = state.admins.find(item => item.id === adminId);

      if (!admin) {
        throwError('admin_not_found', 'Администратор не найден');
      }

      admin.locations = admin.locations.filter(location => location.id !== locationId);
      saveMockState(state);

      return null;
    }

    if (pathname === '/api/superuser/rooms/moderation' && method === 'GET') {
      return {
        items: state.rooms
          .filter(room => room.status === 'pending')
          .map(room => {
            const location = getLocationById(state, room.location_id);

            return {
              id: room.id,
              location_id: room.location_id,
              company_name: location?.company_name || '',
              city: location?.city || '',
              address: location?.address || '',
              title: room.title,
              description: room.description,
              price: room.price,
              capacity: room.capacity,
              available_from: room.available_from,
              available_to: room.available_to,
              images: room.images,
              status: room.status,
              created_by: {
                id: 10,
                username: 'Администратор ABC',
                email: 'admin-abc@mail.com'
              }
            };
          })
      };
    }

    const approveMatch = pathname.match(/^\/api\/superuser\/rooms\/(\d+)\/approve$/);
    if (approveMatch && method === 'POST') {
      const room = getRoomById(state, Number(approveMatch[1]));

      if (!room) {
        throwError('room_not_found', 'Помещение не найдено');
      }

      if (room.status !== 'pending') {
        throwError('cannot_approve_room', 'Можно одобрить только помещение на модерации');
      }

      room.status = 'published';
      room.rejection_reason = null;
      saveMockState(state);

      return {
        id: room.id,
        status: 'published'
      };
    }

    const rejectMatch = pathname.match(/^\/api\/superuser\/rooms\/(\d+)\/reject$/);
    if (rejectMatch && method === 'POST') {
      const room = getRoomById(state, Number(rejectMatch[1]));
      const payload = parseJson(options.body);

      if (!room) {
        throwError('room_not_found', 'Помещение не найдено');
      }

      if (room.status !== 'pending') {
        throwError('cannot_reject_room', 'Можно отклонить только помещение на модерации');
      }

      room.status = 'rejected';
      room.rejection_reason = payload.reason || 'Причина не указана';
      saveMockState(state);

      return {
        id: room.id,
        status: 'rejected',
        rejection_reason: room.rejection_reason
      };
    }

    throwError('mock_not_found', `Mock для ${pathname} не найден`);
  }

  // =========================================================
  // Public methods
  // =========================================================

  function getMe() {
    return apiRequest('/api/me');
  }

  function registerUser(payload) {
    return apiRequest('/api/register', {
      method: 'POST',
      body: JSON.stringify(payload)
    });
  }

  function loginUser(payload) {
    return apiRequest('/api/login', {
      method: 'POST',
      body: JSON.stringify(payload)
    });
  }

  function logoutUser() {
    return apiRequest('/api/logout', {
      method: 'POST'
    });
  }

  function getCities() {
    return apiRequest('/api/cities');
  }

  function getRoomFilters(city) {
    const qs = new URLSearchParams({ city }).toString();
    return apiRequest(`/api/rooms/filters?${qs}`);
  }

  function getRooms(params = {}) {
    const qs = new URLSearchParams(params).toString();
    return apiRequest(`/api/rooms${qs ? `?${qs}` : ''}`);
  }

  function getRoom(id) {
    return apiRequest(`/api/rooms/${encodeURIComponent(id)}`);
  }

  function getRoomAvailability(id, days = 7) {
    return apiRequest(`/api/rooms/${encodeURIComponent(id)}/availability?days=${days}`);
  }

  function createBooking(payload) {
    return apiRequest('/api/bookings', {
      method: 'POST',
      body: JSON.stringify(payload)
    });
  }

  function getMyBookings() {
    return apiRequest('/api/me/bookings');
  }

  function cancelBooking(id) {
    return apiRequest(`/api/bookings/${encodeURIComponent(id)}/cancel`, {
      method: 'POST'
    });
  }

  // =========================================================
  // Superuser methods
  // =========================================================

  function getCompanies() {
    return apiRequest('/api/superuser/companies');
  }

  function createCompany(payload) {
    return apiRequest('/api/superuser/companies', {
      method: 'POST',
      body: JSON.stringify(payload)
    });
  }

  function getLocations(params = {}) {
    const qs = new URLSearchParams(params).toString();
    return apiRequest(`/api/superuser/locations${qs ? `?${qs}` : ''}`);
  }

  function createLocation(payload) {
    return apiRequest('/api/superuser/locations', {
      method: 'POST',
      body: JSON.stringify(payload)
    });
  }

  function getAdmins() {
    return apiRequest('/api/superuser/admins');
  }

  function createAdmin(payload) {
    return apiRequest('/api/superuser/admins', {
      method: 'POST',
      body: JSON.stringify(payload)
    });
  }

  function assignAdminToLocation(adminId, locationId) {
    return apiRequest(`/api/superuser/admins/${encodeURIComponent(adminId)}/locations`, {
      method: 'POST',
      body: JSON.stringify({
        location_id: Number(locationId)
      })
    });
  }

  function removeAdminFromLocation(adminId, locationId) {
    return apiRequest(
      `/api/superuser/admins/${encodeURIComponent(adminId)}/locations/${encodeURIComponent(locationId)}`,
      {
        method: 'DELETE'
      }
    );
  }

  function getModerationRooms() {
    return apiRequest('/api/superuser/rooms/moderation');
  }

  function approveRoom(roomId) {
    return apiRequest(`/api/superuser/rooms/${encodeURIComponent(roomId)}/approve`, {
      method: 'POST'
    });
  }

  function rejectRoom(roomId, reason) {
    return apiRequest(`/api/superuser/rooms/${encodeURIComponent(roomId)}/reject`, {
      method: 'POST',
      body: JSON.stringify({ reason })
    });
  }

  // =========================================================
  // Admin methods
  // =========================================================

  function getAdminLocations() {
    return apiRequest('/api/admin/locations');
  }

  function getAdminRooms(params = {}) {
    const qs = new URLSearchParams(params).toString();
    return apiRequest(`/api/admin/rooms${qs ? `?${qs}` : ''}`);
  }

  function getAdminRoom(id) {
    return apiRequest(`/api/admin/rooms/${encodeURIComponent(id)}`);
  }

  function createAdminRoom(payload) {
    return apiRequest('/api/admin/rooms', {
      method: 'POST',
      body: JSON.stringify(payload)
    });
  }

  function updateAdminRoom(id, payload) {
    return apiRequest(`/api/admin/rooms/${encodeURIComponent(id)}`, {
      method: 'PATCH',
      body: JSON.stringify(payload)
    });
  }

  function submitRoomForModeration(id) {
    return apiRequest(`/api/admin/rooms/${encodeURIComponent(id)}/submit`, {
      method: 'POST'
    });
  }

  function archiveAdminRoom(id, mode) {
    return apiRequest(`/api/admin/rooms/${encodeURIComponent(id)}/archive`, {
      method: 'POST',
      body: JSON.stringify({ mode })
    });
  }

  function getAdminBookings(params = {}) {
    const qs = new URLSearchParams(params).toString();
    return apiRequest(`/api/admin/bookings${qs ? `?${qs}` : ''}`);
  }

  function cancelAdminBooking(id) {
    return apiRequest(`/api/admin/bookings/${encodeURIComponent(id)}/cancel`, {
      method: 'POST'
    });
  }

  // =========================================================
  // Mock helpers
  // =========================================================

  function getMockUser() {
    const users = {
      user: {
        id: 100,
        username: 'Тестовый пользователь',
        email: 'user@mail.com',
        role: 'user'
      },
      admin: {
        id: 10,
        username: 'Администратор ABC',
        email: 'admin-abc@mail.com',
        role: 'admin'
      },
      superuser: {
        id: 1,
        username: 'Суперпользователь',
        email: 'superuser@mail.com',
        role: 'superuser'
      }
    };

    return users[MOCK_ROLE] || users.user;
  }

  function getMockState() {
    const saved = localStorage.getItem(MOCK_STORAGE_KEY);

    if (saved) {
      try {
        return JSON.parse(saved);
      } catch {
        localStorage.removeItem(MOCK_STORAGE_KEY);
      }
    }

    const initial = createInitialMockState();
    saveMockState(initial);
    return initial;
  }

  function saveMockState(state) {
    localStorage.setItem(MOCK_STORAGE_KEY, JSON.stringify(state));
  }

  function createInitialMockState() {
    const now = new Date().toISOString();

    return {
      nextCompanyId: 6,
      nextLocationId: 6,
      nextAdminId: 15,
      nextRoomId: 30,
      nextBookingId: 200,

      companies: [
        {
          id: 1,
          name: 'ABC Coworking',
          description: 'Сеть коворкингов',
          locations_count: 2
        },
        {
          id: 2,
          name: 'Office Rent',
          description: 'Аренда офисных пространств',
          locations_count: 1
        },
        {
          id: 3,
          name: 'Meeting Space',
          description: 'Переговорные комнаты',
          locations_count: 1
        },
        {
          id: 4,
          name: 'Business Rooms',
          description: 'Бизнес-площадки',
          locations_count: 1
        },
        {
          id: 5,
          name: 'WorkPoint',
          description: 'Рабочие пространства',
          locations_count: 0
        }
      ],

      locations: [
        {
          id: 1,
          company_id: 1,
          company_name: 'ABC Coworking',
          city: 'Москва',
          address: 'Москва, Тверская 10',
          lat: 55.7558,
          lng: 37.6173,
          timezone: 'Europe/Moscow',
          rooms_count: 3
        },
        {
          id: 2,
          company_id: 1,
          company_name: 'ABC Coworking',
          city: 'Москва',
          address: 'Москва, Арбат 15',
          lat: 55.7522,
          lng: 37.6156,
          timezone: 'Europe/Moscow',
          rooms_count: 2
        },
        {
          id: 3,
          company_id: 2,
          company_name: 'Office Rent',
          city: 'Санкт-Петербург',
          address: 'Невский проспект 20',
          lat: 59.9343,
          lng: 30.3351,
          timezone: 'Europe/Moscow',
          rooms_count: 1
        },
        {
          id: 4,
          company_id: 3,
          company_name: 'Meeting Space',
          city: 'Казань',
          address: 'Казань, Баумана 7',
          lat: 55.7961,
          lng: 49.1064,
          timezone: 'Europe/Moscow',
          rooms_count: 1
        },
        {
          id: 5,
          company_id: 4,
          company_name: 'Business Rooms',
          city: 'Екатеринбург',
          address: 'Екатеринбург, Ленина 1',
          lat: 56.8389,
          lng: 60.6057,
          timezone: 'Asia/Yekaterinburg',
          rooms_count: 1
        }
      ],

      admins: [
        {
          id: 10,
          username: 'Администратор ABC',
          email: 'admin-abc@mail.com',
          role: 'admin',
          locations: [
            {
              id: 1,
              address: 'Москва, Тверская 10',
              company_name: 'ABC Coworking'
            },
            {
              id: 2,
              address: 'Москва, Арбат 15',
              company_name: 'ABC Coworking'
            }
          ]
        },
        {
          id: 11,
          username: 'Администратор Office',
          email: 'admin-office@mail.com',
          role: 'admin',
          locations: [
            {
              id: 3,
              address: 'Невский проспект 20',
              company_name: 'Office Rent'
            }
          ]
        },
        {
          id: 12,
          username: 'Администратор Meeting',
          email: 'admin-meeting@mail.com',
          role: 'admin',
          locations: [
            {
              id: 4,
              address: 'Казань, Баумана 7',
              company_name: 'Meeting Space'
            }
          ]
        },
        {
          id: 13,
          username: 'Администратор Business',
          email: 'admin-business@mail.com',
          role: 'admin',
          locations: [
            {
              id: 5,
              address: 'Екатеринбург, Ленина 1',
              company_name: 'Business Rooms'
            }
          ]
        }
      ],

      rooms: [
        createMockRoom(15, 1, 'Переговорная на 8 человек', 'draft', now, 1500, 8),
        createMockRoom(16, 1, 'Большой конференц-зал', 'pending', now, 3500, 20),
        createMockRoom(17, 1, 'Опубликованная переговорная', 'published', now, 1500, 8),
        createMockRoom(18, 2, 'Малая переговорная', 'rejected', now, 900, 4, 'Добавьте фотографии помещения'),
        createMockRoom(19, 2, 'Архивная переговорная', 'archived', now, 1000, 6),
        createMockRoom(20, 3, 'Комната в Санкт-Петербурге', 'published', now, 2200, 10),
        createMockRoom(21, 4, 'Казанская переговорная', 'published', now, 1200, 6),
        createMockRoom(22, 5, 'Екатеринбург конференц-зал', 'pending', now, 3000, 18)
      ],

      bookings: [
        {
          id: 101,
          room_id: 17,
          room_title: 'Опубликованная переговорная',
          location_id: 1,
          location_address: 'Москва, Тверская 10',
          user_id: 5,
          user_email: 'user@mail.com',
          user_username: 'Иван',
          date: getDatePlusDays(2),
          start_time: '10:00',
          end_time: '12:00',
          total_price: 3000,
          status: 'booked'
        },
        {
          id: 102,
          room_id: 17,
          room_title: 'Опубликованная переговорная',
          location_id: 1,
          location_address: 'Москва, Тверская 10',
          user_id: 6,
          user_email: 'petrov@mail.com',
          user_username: 'Пётр',
          date: getDatePlusDays(0),
          start_time: '14:00',
          end_time: '16:00',
          total_price: 3000,
          status: 'in_use'
        },
        {
          id: 103,
          room_id: 20,
          room_title: 'Комната в Санкт-Петербурге',
          location_id: 3,
          location_address: 'Невский проспект 20',
          user_id: 7,
          user_email: 'anna@mail.com',
          user_username: 'Анна',
          date: getDatePlusDays(-1),
          start_time: '09:00',
          end_time: '10:00',
          total_price: 2200,
          status: 'finished'
        },
        {
          id: 104,
          room_id: 17,
          room_title: 'Опубликованная переговорная',
          location_id: 1,
          location_address: 'Москва, Тверская 10',
          user_id: 8,
          user_email: 'canceled@mail.com',
          user_username: 'Олег',
          date: getDatePlusDays(4),
          start_time: '16:00',
          end_time: '17:00',
          total_price: 1500,
          status: 'canceled'
        }
      ]
    };
  }

  function createMockRoom(id, locationId, title, status, createdAt, price, capacity, rejectionReason = null) {
    return {
      id,
      location_id: locationId,
      title,
      description: 'Описание помещения. Есть экран, доска, Wi-Fi и удобные кресла.',
      price,
      capacity,
      available_from: '09:00',
      available_to: '21:00',
      images: ['/shared/placeholders/room-placeholder.svg'],
      status,
      rejection_reason: rejectionReason,
      created_at: createdAt,
      archive: {
        can_archive_now: true,
        has_active_or_future_bookings: false,
        booking_disabled: status === 'archived',
        scheduled_for: null
      }
    };
  }

  function getVisibleAdminLocations(state) {
    if (MOCK_ROLE === 'superuser') {
      return state.locations;
    }

    const user = getMockUser();

    if (user.role !== 'admin') {
      return [];
    }

    const admin = state.admins.find(item => item.id === user.id);

    if (!admin) {
      return [];
    }

    const assignedLocationIds = admin.locations.map(location => location.id);

    return state.locations.filter(location => assignedLocationIds.includes(location.id));
  }

  function toAdminRoomListItem(room) {
    return {
      id: room.id,
      location_id: room.location_id,
      title: room.title,
      price: room.price,
      capacity: room.capacity,
      status: room.status,
      rejection_reason: room.rejection_reason,
      created_at: room.created_at
    };
  }

  function getRoomById(state, id) {
    return state.rooms.find(room => Number(room.id) === Number(id));
  }

  function getLocationById(state, id) {
    return state.locations.find(location => Number(location.id) === Number(id));
  }

  function getBookingById(state, id) {
    return state.bookings.find(booking => Number(booking.id) === Number(id));
  }

  function refreshRoomArchiveInfo(state, room) {
    const hasActiveOrFutureBookings = state.bookings.some(booking => {
      if (Number(booking.room_id) !== Number(room.id)) {
        return false;
      }

      if (booking.status !== 'booked') {
        return false;
      }

      return booking.date >= getDatePlusDays(0);
    });

    room.archive = room.archive || {};
    room.archive.has_active_or_future_bookings = hasActiveOrFutureBookings;
    room.archive.can_archive_now = !hasActiveOrFutureBookings && room.status !== 'archived';

    if (room.status === 'archived') {
      room.archive.booking_disabled = true;
      room.archive.can_archive_now = false;
    }
  }

  function normalizeImages(images) {
    if (Array.isArray(images) && images.length > 0) {
      return images;
    }

    return ['/shared/placeholders/room-placeholder.svg'];
  }

  function parseJson(raw) {
    try {
      return raw ? JSON.parse(raw) : {};
    } catch {
      return {};
    }
  }

  function throwError(code, message) {
    throw {
      code,
      message
    };
  }

  function getDatePlusDays(days) {
    const d = new Date();
    d.setDate(d.getDate() + days);

    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');

    return `${y}-${m}-${day}`;
  }

  function getDatePlusDaysIso(days) {
    const d = new Date();
    d.setDate(d.getDate() + days);
    return d.toISOString();
  }

  function addOneHour(time) {
    const [h, m] = String(time).split(':').map(Number);
    const nextHour = Number.isFinite(h) ? h + 1 : 11;
    const minutes = Number.isFinite(m) ? m : 0;

    return `${String(nextHour).padStart(2, '0')}:${String(minutes).padStart(2, '0')}`;
  }

  window.Api = {
    apiRequest,

    getMe,
    registerUser,
    loginUser,
    logoutUser,

    getCities,
    getRoomFilters,
    getRooms,
    getRoom,
    getRoomAvailability,

    createBooking,
    getMyBookings,
    cancelBooking,

    getCompanies,
    createCompany,
    getLocations,
    createLocation,
    getAdmins,
    createAdmin,
    assignAdminToLocation,
    removeAdminFromLocation,
    getModerationRooms,
    approveRoom,
    rejectRoom,

    getAdminLocations,
    getAdminRooms,
    getAdminRoom,
    createAdminRoom,
    updateAdminRoom,
    submitRoomForModeration,
    archiveAdminRoom,
    getAdminBookings,
    cancelAdminBooking
  };
})();