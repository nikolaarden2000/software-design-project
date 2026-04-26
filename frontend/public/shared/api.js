(function () {
  'use strict';

  const USE_MOCKS = false;

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
    await new Promise(resolve => setTimeout(resolve, 200));

    if (path === '/api/me') {
      return {
        authenticated: true,
        user: {
          id: 1,
          username: 'Тестовый пользователь',
          email: 'test@mail.com',
          role: 'user'
        }
      };
    }

    if (path.startsWith('/api/rooms/filters')) {
      return {
        city: 'Москва',
        companies: [
          'ABC Coworking',
          'Office Rent',
          'Meeting Space'
        ],
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

    if (path.startsWith('/api/rooms?')) {
      return {
        items: [
          {
            id: 1,
            title: 'Переговорная на 8 человек',
            company: 'ABC Coworking',
            address: 'Москва, Тверская 10',
            capacity: 8,
            image: '/shared/placeholders/room-placeholder.svg',
            price: 1500
          },
          {
            id: 2,
            title: 'Большой конференц-зал',
            company: 'Office Rent',
            address: 'Москва, Арбат 15',
            capacity: 20,
            image: '/shared/placeholders/room-placeholder.svg',
            price: 3500
          },
          {
            id: 3,
            title: 'Малая переговорная',
            company: 'Meeting Space',
            address: 'Москва, Ленина 5',
            capacity: 4,
            image: '/shared/placeholders/room-placeholder.svg',
            price: 900
          }
        ],
        pagination: {
          limit: 100,
          next_after_id: null,
          has_more: false
        }
      };
    }

    const roomMatch = path.match(/^\/api\/rooms\/(\d+)$/);
    if (roomMatch) {
      const id = Number(roomMatch[1]);

      return {
        id,
        title: id === 2 ? 'Большой конференц-зал' : 'Переговорная на 8 человек',
        company: id === 2 ? 'Office Rent' : 'ABC Coworking',
        address: id === 2 ? 'Москва, Арбат 15' : 'Москва, Тверская 10',
        images: [
          '/shared/placeholders/room-placeholder.svg'
        ],
        price: id === 2 ? 3500 : 1500,
        currency: 'RUB',
        capacity: id === 2 ? 20 : 8,
        max_capacity: id === 2 ? 20 : 8,
        available_from: '09:00',
        available_to: '21:00',
        description: 'Светлая переговорная с экраном, доской и удобными креслами.',
        lat: 55.7558,
        lng: 37.6173
      };
    }

    const availabilityMatch = path.match(/^\/api\/rooms\/(\d+)\/availability/);
    if (availabilityMatch) {
      return {
        room_id: Number(availabilityMatch[1]),
        dates: [
          {
            date: getDatePlusDays(0),
            available_times: ['10:00', '11:00', '12:00', '15:00']
          },
          {
            date: getDatePlusDays(1),
            available_times: ['09:00', '13:00', '14:00', '18:00']
          },
          {
            date: getDatePlusDays(2),
            available_times: ['10:00', '16:00', '17:00']
          }
        ]
      };
    }

    if (path === '/api/bookings' && options.method === 'POST') {
      return {
        id: 101,
        room_id: 1,
        date: getDatePlusDays(0),
        start_time: '10:00',
        end_time: '12:00',
        status: 'booked'
      };
    }

    if (path === '/api/me/bookings') {
      return {
        items: [
          {
            id: 101,
            room_id: 1,
            image_url: '/shared/placeholders/room-placeholder.svg',
            title: 'Переговорная на 8 человек',
            date: getDatePlusDays(1),
            start_time: '10:00',
            end_time: '12:00',
            total_price: 3000,
            status: 'booked'
          },
          {
            id: 102,
            room_id: 2,
            image_url: '/shared/placeholders/room-placeholder.svg',
            title: 'Большой конференц-зал',
            date: getDatePlusDays(-1),
            start_time: '13:00',
            end_time: '15:00',
            total_price: 7000,
            status: 'finished'
          }
        ]
      };
    }

    const cancelMatch = path.match(/^\/api\/bookings\/(\d+)\/cancel$/);
    if (cancelMatch) {
      return {
        id: Number(cancelMatch[1]),
        status: 'canceled'
      };
    }

    if (path === '/api/login') {
      return {
        user: {
          id: 1,
          username: 'Тестовый пользователь',
          email: 'test@mail.com',
          role: 'user'
        }
      };
    }

    if (path === '/api/register') {
      return {
        id: 2,
        username: 'new-user',
        email: 'new@mail.com',
        role: 'user'
      };
    }

    if (path === '/api/logout') {
      return null;
    }

    throw {
      code: 'mock_not_found',
      message: `Mock для ${path} не найден`
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

  function getRoomFilters(city) {
    const qs = new URLSearchParams({ city }).toString();
    return apiRequest(`/api/rooms/filters?${qs}`);
  }

  function getRooms(params) {
    const qs = new URLSearchParams(params).toString();
    return apiRequest(`/api/rooms?${qs}`);
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

  window.Api = {
    apiRequest,
    getMe,
    registerUser,
    loginUser,
    logoutUser,
    getRoomFilters,
    getRooms,
    getRoom,
    getRoomAvailability,
    createBooking,
    getMyBookings,
    cancelBooking
  };
})();