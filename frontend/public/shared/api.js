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
      id: 10,
      username: 'Администратор ABC',
      email: 'admin-abc@mail.com',
      role: 'admin'
    }
  };
}

if (path === '/api/superuser/companies') {
  if (options.method === 'POST') {
    const payload = JSON.parse(options.body || '{}');

    return {
      id: Date.now(),
      name: payload.name,
      description: payload.description || '',
      locations_count: 0
    };
  }

  return {
    items: [
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
      }
    ]
  };
}
if (path === '/api/admin/locations') {
  return {
    items: [
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
        rooms_count: 1
      }
    ]
  };
}

if (path.startsWith('/api/admin/rooms') && !path.includes('/submit')) {
  if (options.method === 'POST') {
    const payload = JSON.parse(options.body || '{}');

    return {
      id: Date.now(),
      location_id: Number(payload.location_id),
      title: payload.title,
      price: Number(payload.price),
      capacity: Number(payload.capacity),
      status: 'draft',
      rejection_reason: null,
      created_at: new Date().toISOString()
    };
  }

  return {
    items: [
      {
        id: 1,
        location_id: 1,
        title: 'Переговорная на 8 человек',
        price: 1500,
        capacity: 8,
        status: 'draft',
        rejection_reason: null,
        created_at: new Date().toISOString()
      },
      {
        id: 2,
        location_id: 1,
        title: 'Большой конференц-зал',
        price: 3500,
        capacity: 20,
        status: 'pending',
        rejection_reason: null,
        created_at: new Date().toISOString()
      },
      {
        id: 3,
        location_id: 2,
        title: 'Малая переговорная',
        price: 900,
        capacity: 4,
        status: 'rejected',
        rejection_reason: 'Добавьте фотографии помещения',
        created_at: new Date().toISOString()
      }
    ]
  };
}

const submitAdminRoomMatch = path.match(/^\/api\/admin\/rooms\/(\d+)\/submit$/);
if (submitAdminRoomMatch && options.method === 'POST') {
  return {
    id: Number(submitAdminRoomMatch[1]),
    status: 'pending'
  };
}

if (path.startsWith('/api/admin/bookings')) {
  return {
    items: [
      {
        id: 101,
        room_id: 1,
        room_title: 'Переговорная на 8 человек',
        location_id: 1,
        location_address: 'Москва, Тверская 10',
        user_id: 5,
        user_email: 'user@mail.com',
        user_username: 'Иван',
        date: getDatePlusDays(1),
        start_time: '10:00',
        end_time: '12:00',
        total_price: 3000,
        status: 'booked'
      },
      {
        id: 102,
        room_id: 2,
        room_title: 'Большой конференц-зал',
        location_id: 1,
        location_address: 'Москва, Тверская 10',
        user_id: 6,
        user_email: 'petrov@mail.com',
        user_username: 'Пётр',
        date: getDatePlusDays(0),
        start_time: '14:00',
        end_time: '16:00',
        total_price: 7000,
        status: 'in_use'
      },
      {
        id: 103,
        room_id: 3,
        room_title: 'Малая переговорная',
        location_id: 2,
        location_address: 'Москва, Арбат 15',
        user_id: 7,
        user_email: 'anna@mail.com',
        user_username: 'Анна',
        date: getDatePlusDays(-1),
        start_time: '09:00',
        end_time: '10:00',
        total_price: 900,
        status: 'finished'
      }
    ]
  };
}

const cancelAdminBookingMatch = path.match(/^\/api\/admin\/bookings\/(\d+)\/cancel$/);
if (cancelAdminBookingMatch && options.method === 'POST') {
  return {
    id: Number(cancelAdminBookingMatch[1]),
    status: 'canceled'
  };
}

if (path.startsWith('/api/superuser/locations')) {
  if (options.method === 'POST') {
    const payload = JSON.parse(options.body || '{}');

    return {
      id: Date.now(),
      company_id: Number(payload.company_id),
      company_name: 'Новая компания',
      city: payload.city,
      address: payload.address,
      lat: Number(payload.lat),
      lng: Number(payload.lng),
      timezone: payload.timezone
    };
  }

  return {
    items: [
      {
        id: 1,
        company_id: 1,
        company_name: 'ABC Coworking',
        city: 'Москва',
        address: 'Москва, Тверская 10',
        lat: 55.7558,
        lng: 37.6173,
        timezone: 'Europe/Moscow'
      },
      {
        id: 2,
        company_id: 2,
        company_name: 'Office Rent',
        city: 'Санкт-Петербург',
        address: 'Невский проспект 20',
        lat: 59.9343,
        lng: 30.3351,
        timezone: 'Europe/Moscow'
      }
    ]
  };
}

if (path === '/api/superuser/admins') {
  if (options.method === 'POST') {
    const payload = JSON.parse(options.body || '{}');

    return {
      id: Date.now(),
      username: payload.username,
      email: payload.email,
      role: 'admin',
      locations: []
    };
  }

  return {
    items: [
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
          }
        ]
      }
    ]
  };
}

const assignMatch = path.match(/^\/api\/superuser\/admins\/(\d+)\/locations$/);
if (assignMatch && options.method === 'POST') {
  const payload = JSON.parse(options.body || '{}');

  return {
    admin_id: Number(assignMatch[1]),
    location_id: Number(payload.location_id)
  };
}

if (path === '/api/superuser/rooms/moderation') {
  return {
    items: [
      {
        id: 101,
        location_id: 1,
        company_name: 'ABC Coworking',
        city: 'Москва',
        address: 'Москва, Тверская 10',
        title: 'Новая переговорная',
        description: 'Комната с экраном, доской и кондиционером.',
        price: 1600,
        capacity: 8,
        available_from: '09:00',
        available_to: '21:00',
        images: [
          '/shared/placeholders/room-placeholder.svg'
        ],
        status: 'pending',
        created_by: {
          id: 10,
          username: 'Администратор ABC',
          email: 'admin-abc@mail.com'
        }
      }
    ]
  };
}

const approveMatch = path.match(/^\/api\/superuser\/rooms\/(\d+)\/approve$/);
if (approveMatch && options.method === 'POST') {
  return {
    id: Number(approveMatch[1]),
    status: 'published'
  };
}

const rejectMatch = path.match(/^\/api\/superuser\/rooms\/(\d+)\/reject$/);
if (rejectMatch && options.method === 'POST') {
  const payload = JSON.parse(options.body || '{}');

  return {
    id: Number(rejectMatch[1]),
    status: 'rejected',
    rejection_reason: payload.reason || 'Причина не указана'
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

function getAdminBookings(params = {}) {
  const qs = new URLSearchParams(params).toString();
  return apiRequest(`/api/admin/bookings${qs ? `?${qs}` : ''}`);
}

function cancelAdminBooking(id, reason) {
  return apiRequest(`/api/admin/bookings/${encodeURIComponent(id)}/cancel`, {
    method: 'POST',
    body: JSON.stringify({
      reason
    })
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
  cancelBooking,

  getCompanies,
  createCompany,
  getLocations,
  createLocation,
  getAdmins,
  createAdmin,
  assignAdminToLocation,
  getModerationRooms,
  approveRoom,
  rejectRoom,

  getAdminLocations,
  getAdminRooms,
  getAdminRoom,
  createAdminRoom,
  updateAdminRoom,
  submitRoomForModeration,
  getAdminBookings,
  cancelAdminBooking
};
})();