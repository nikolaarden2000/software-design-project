
const adminModule = require('./admin.js');

describe('admin.js', () => {
  async function flushPromises() {
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
  }

  function buildBaseDOM() {
    document.body.innerHTML = `
      <button id="logoutBtn">Выйти</button>
      <section id="accessDenied" class="hidden">Нет доступа</section>
      <section id="adminContent" class="hidden">
        <div id="adminRoot"></div>
      </section>
    `;
  }

  function buildRoomFormDOM() {
    document.body.innerHTML = `
      <div id="adminRoot"></div>
      <input id="roomLocationId" value="1" />
      <input id="roomTitle" value="Переговорная" />
      <textarea id="roomDescription">Описание комнаты</textarea>
      <input id="roomPrice" value="1500" />
      <input id="roomCapacity" value="8" />
      <input id="roomAvailableFrom" value="09:00" />
      <input id="roomAvailableTo" value="21:00" />
      <textarea id="roomImages">https://img/1.png, https://img/2.png</textarea>
    `;
  }

  function makeApi(overrides = {}) {
    return {
      getMe: jest.fn().mockResolvedValue({
        authenticated: true,
        user: { role: 'admin' }
      }),
      getAdminLocations: jest.fn().mockResolvedValue({ items: [] }),
      getAdminRooms: jest.fn().mockResolvedValue({ items: [] }),
      getAdminRoom: jest.fn().mockResolvedValue({
        id: 101,
        location_id: 1,
        title: 'Комната 101',
        description: 'Описание',
        price: 1000,
        capacity: 10,
        available_from: '09:00',
        available_to: '21:00',
        images: ['/img/1.png'],
        status: 'draft',
        rejection_reason: ''
      }),
      createAdminRoom: jest.fn().mockResolvedValue({ id: 500 }),
      updateAdminRoom: jest.fn().mockResolvedValue({}),
      submitRoomForModeration: jest.fn().mockResolvedValue({}),
      getAdminBookings: jest.fn().mockResolvedValue({ items: [] }),
      cancelAdminBooking: jest.fn().mockResolvedValue({}),
      archiveAdminRoom: jest.fn().mockResolvedValue({}),
      logoutUser: jest.fn().mockResolvedValue({}),
      ...overrides
    };
  }

  beforeEach(() => {
    jest.clearAllMocks();

    global.alert = jest.fn();
    global.confirm = jest.fn(() => true);

    global.console = {
      log: jest.fn(),
      warn: jest.fn(),
      error: jest.fn()
    };

    delete window.location;
    window.location = {
      href: '',
      pathname: '/admin',
      search: ''
    };

    window.Api = makeApi();

    buildBaseDOM();
  });

  test('escapeHtml: экранирует HTML-символы', () => {
    // Техника тест-дизайна: классы эквивалентности
    expect(adminModule.escapeHtml('<script>"x"&\'y\'</script>'))
      .toBe('&lt;script&gt;&quot;x&quot;&amp;&#039;y&#039;&lt;/script&gt;');
  });

  test('escapeAttr: дополнительно экранирует обратную кавычку', () => {
    // Техника тест-дизайна: классы эквивалентности
    expect(adminModule.escapeAttr('`test`')).toBe('&#096;test&#096;');
  });

  test('getRoomStatusLabel: возвращает русскую подпись для draft', () => {
    // Техника тест-дизайна: классы эквивалентности
    expect(adminModule.getRoomStatusLabel('draft')).toBe('Черновик');
  });


  test('getBookingStatusLabel: возвращает русскую подпись для booked', () => {
    // Техника тест-дизайна: классы эквивалентности
    expect(adminModule.getBookingStatusLabel('booked')).toBe('Забронировано');
  });

  test('formatDateTime: при пустом значении возвращает прочерк', () => {
    // Техника тест-дизайна: граничные условия
    expect(adminModule.formatDateTime('')).toBe('—');
  });

  test('showAccessDenied: показывает блок отказа и скрывает контент', () => {
    // Техника тест-дизайна: переходы состояний
    adminModule.showAccessDenied();

    expect(document.getElementById('accessDenied').classList.contains('hidden')).toBe(false);
    expect(document.getElementById('adminContent').classList.contains('hidden')).toBe(true);
  });

  test('showContent: скрывает отказ в доступе и показывает контент', () => {
    // Техника тест-дизайна: переходы состояний
    adminModule.showContent();

    expect(document.getElementById('accessDenied').classList.contains('hidden')).toBe(true);
    expect(document.getElementById('adminContent').classList.contains('hidden')).toBe(false);
  });

  test('validateRoomPayload: без location_id возвращает false и показывает alert', () => {
    // Техника тест-дизайна: классы эквивалентности
    const result = adminModule.validateRoomPayload({
      location_id: 0,
      title: 'Зал',
      description: 'Описание',
      price: 1000,
      capacity: 10
    });

    expect(result).toBe(false);
    expect(global.alert).toHaveBeenCalledWith('Не указан location_id');
  });

  test('validateRoomPayload: без title/description возвращает false', () => {
    // Техника тест-дизайна: классы эквивалентности
    const result = adminModule.validateRoomPayload({
      location_id: 1,
      title: '',
      description: '',
      price: 1000,
      capacity: 10
    });

    expect(result).toBe(false);
    expect(global.alert).toHaveBeenCalledWith('Введите название и описание');
  });

  test('validateRoomPayload: price <= 0 и capacity <= 0 возвращает false', () => {
    // Техника тест-дизайна: граничные условия
    const result = adminModule.validateRoomPayload({
      location_id: 1,
      title: 'Зал',
      description: 'Описание',
      price: 0,
      capacity: 0
    });

    expect(result).toBe(false);
    expect(global.alert).toHaveBeenCalledWith('Цена и вместимость должны быть больше нуля');
  });

  test('validateRoomPayload: корректный payload проходит валидацию', () => {
    // Техника тест-дизайна: классы эквивалентности
    const result = adminModule.validateRoomPayload({
      location_id: 1,
      title: 'Зал',
      description: 'Описание',
      price: 1000,
      capacity: 10
    });

    expect(result).toBe(true);
    expect(global.alert).not.toHaveBeenCalled();
  });

  test('readRoomForm: корректно читает форму и разбирает список изображений', () => {
    // Техника тест-дизайна: сценарий использования
    buildRoomFormDOM();

    const payload = adminModule.readRoomForm();

    expect(payload).toEqual({
      location_id: 1,
      title: 'Переговорная',
      description: 'Описание комнаты',
      price: 1500,
      capacity: 8,
      available_from: '09:00',
      available_to: '21:00',
      images: ['https://img/1.png', 'https://img/2.png']
    });
  });

  test('readRoomForm: если изображения не указаны, подставляет placeholder', () => {
    // Техника тест-дизайна: классы эквивалентности
    buildRoomFormDOM();
    document.getElementById('roomImages').value = '';

    const payload = adminModule.readRoomForm();

    expect(payload.images).toEqual(['/shared/placeholders/room-placeholder.svg']);
  });

  test('renderRoomsTable: при пустом списке показывает сообщение', () => {
    // Техника тест-дизайна: классы эквивалентности
    document.getElementById('adminRoot').innerHTML = `<div id="roomsTable"></div>`;

    adminModule.renderRoomsTable([]);

    expect(document.getElementById('roomsTable').textContent).toContain('Помещений пока нет');
  });

  test('renderRoomsTable: отрисовывает строки помещений и клик ведёт на страницу комнаты', () => {
    // Техника тест-дизайна: сценарий использования
    document.getElementById('adminRoot').innerHTML = `<div id="roomsTable"></div>`;

    adminModule.renderRoomsTable([
      {
        id: 101,
        title: 'Комната 101',
        capacity: 10,
        price: 1500,
        status: 'draft',
        rejection_reason: '',
        created_at: '2026-04-30T10:00:00Z'
      }
    ]);

    const row = document.querySelector('[data-room-id="101"]');
    expect(row).not.toBeNull();
    expect(document.getElementById('roomsTable').textContent).toContain('Комната 101');

    row.click();
    expect(window.location.href).toBe('/admin/room/101');
  });

  test('renderBookingsTable: при пустом списке показывает сообщение', () => {
    // Техника тест-дизайна: классы эквивалентности
    document.getElementById('adminRoot').innerHTML = `<div id="bookingsTable"></div>`;

    adminModule.renderBookingsTable([]);

    expect(document.getElementById('bookingsTable').textContent).toContain('Бронирований пока нет');
  });

  test('renderBookingsTable: сортирует бронирования по дате и времени', () => {
    // Техника тест-дизайна: сценарий использования
    document.getElementById('adminRoot').innerHTML = `<div id="bookingsTable"></div>`;

    adminModule.renderBookingsTable([
      {
        id: 2,
        user_username: 'u2',
        user_email: 'u2@test.com',
        date: '2026-05-02',
        start_time: '11:00',
        end_time: '12:00',
        total_price: 2000,
        status: 'finished'
      },
      {
        id: 1,
        user_username: 'u1',
        user_email: 'u1@test.com',
        date: '2026-05-01',
        start_time: '10:00',
        end_time: '11:00',
        total_price: 1000,
        status: 'booked'
      }
    ]);

    const rows = document.querySelectorAll('#bookingsTable tbody tr');
    expect(rows[0].textContent).toContain('1');
    expect(rows[1].textContent).toContain('2');
  });

  test('renderArchiveBlock: для archived-комнаты показывает сообщение, что помещение уже архивировано', () => {
    // Техника тест-дизайна: классы эквивалентности
    const html = adminModule.renderArchiveBlock({
      status: 'archived'
    });

    expect(html).toContain('Помещение уже архивировано');
  });

  test('renderArchiveBlock: для помещения с активными бронированиями показывает warning', () => {
    // Техника тест-дизайна: классы эквивалентности
    const html = adminModule.renderArchiveBlock({
      status: 'published',
      archive: {
        has_active_or_future_bookings: true,
        can_archive_now: false
      }
    });

    expect(html).toContain('У помещения есть активные или будущие бронирования');
    expect(html).toContain('Архивировать сейчас');
  });

  test('renderArchiveBlock: для помещения с запланированным архивированием показывает scheduled-сообщение', () => {
    // Техника тест-дизайна: таблица решений
    const html = adminModule.renderArchiveBlock({
      status: 'published',
      booking_disabled: true,
      archive_scheduled_for: '2026-05-01T15:00:00Z',
      archive: {}
    });

    expect(html).toContain('Помещение ожидает архивирования');
    expect(html).toContain('Новые бронирования отключены');
  });

  test('renderError: показывает страницу ошибки', () => {
    // Техника тест-дизайна: сценарий использования
    adminModule.renderError('Страница не найдена');

    expect(document.getElementById('adminRoot').textContent).toContain('Страница не найдена');
    expect(document.getElementById('adminRoot').textContent).toContain('В админ-панель');
  });


  test('routeAdminPage: для /admin/location/15 рендерит страницу локации', async () => {
  // Техника тест-дизайна: таблица решений
  window.location.pathname = '/admin/location/15';
  window.Api = makeApi({
    getAdminLocations: jest.fn().mockResolvedValue({
      items: [
        {
          id: 15,
          company_name: 'Компания X',
          city: 'Москва',
          address: 'Адрес X',
          rooms_count: 0,
          timezone: 'Europe/Moscow'
        }
      ]
    }),
    getAdminRooms: jest.fn().mockResolvedValue({ items: [] })
  });

  await adminModule.routeAdminPage();

  expect(document.getElementById('adminRoot').textContent).toContain('Компания X');
  expect(document.getElementById('adminRoot').textContent).toContain('Помещения локации');
  expect(window.Api.getAdminRooms).toHaveBeenCalledWith({ location_id: 15 });
});


  test('routeAdminPage: для неизвестного маршрута показывает ошибку', async () => {
  // Техника тест-дизайна: классы эквивалентности
  window.location.pathname = '/admin/unknown';

  await adminModule.routeAdminPage();

  expect(document.getElementById('adminRoot').textContent).toContain('Ошибка');
  expect(document.getElementById('adminRoot').textContent).toContain('Страница не найдена');
});

  test('bindLogout: успешный logout вызывает Api.logoutUser и ведёт на главную', async () => {
    // Техника тест-дизайна: сценарий использования
    adminModule.bindLogout();

    document.getElementById('logoutBtn').click();
    await flushPromises();

    expect(window.Api.logoutUser).toHaveBeenCalled();
    expect(window.location.href).toBe('/');
  });

  test('bindLogout: ошибка logout показывает alert', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    window.Api = makeApi({
      logoutUser: jest.fn().mockRejectedValue(new Error('Ошибка выхода'))
    });

    adminModule.bindLogout();

    document.getElementById('logoutBtn').click();
    await flushPromises();

    expect(global.alert).toHaveBeenCalledWith('Ошибка выхода');
    expect(console.error).toHaveBeenCalled();
  });

  test('loadLocations: загружает список локаций и возвращает его', async () => {
    // Техника тест-дизайна: классы эквивалентности
    window.Api = makeApi({
      getAdminLocations: jest.fn().mockResolvedValue({
        items: [{ id: 1 }, { id: 2 }]
      })
    });

    const items = await adminModule.loadLocations();

    expect(window.Api.getAdminLocations).toHaveBeenCalled();
    expect(items).toEqual([{ id: 1 }, { id: 2 }]);
  });

 test('loadRoomsForLocation: без выбранного статуса вызывает getAdminRooms только с location_id и рендерит таблицу', async () => {
  // Техника тест-дизайна: классы эквивалентности
  document.getElementById('adminRoot').innerHTML = `
    <select id="statusFilter">
      <option value="">Все статусы</option>
    </select>
    <div id="roomsTable"></div>
  `;

  window.Api = makeApi({
    getAdminRooms: jest.fn().mockResolvedValue({
      items: [
        {
          id: 5,
          title: 'Комната 5',
          capacity: 4,
          price: 900,
          status: 'draft',
          rejection_reason: '',
          created_at: '2026-04-30T10:00:00Z'
        }
      ]
    })
  });

  await adminModule.loadRoomsForLocation(5);

  expect(window.Api.getAdminRooms).toHaveBeenCalledWith({
    location_id: 5
  });
  expect(document.getElementById('roomsTable').textContent).toContain('Комната 5');
});
  test('initAdminPage: для admin-пользователя показывает контент и рендерит главную страницу', async () => {
    // Техника тест-дизайна: сценарий использования
    window.location.pathname = '/admin';
    window.Api = makeApi({
      getMe: jest.fn().mockResolvedValue({
        authenticated: true,
        user: { role: 'admin' }
      }),
      getAdminLocations: jest.fn().mockResolvedValue({ items: [] })
    });

    await adminModule.initAdminPage();

    expect(document.getElementById('accessDenied').classList.contains('hidden')).toBe(true);
    expect(document.getElementById('adminContent').classList.contains('hidden')).toBe(false);
    expect(document.getElementById('adminRoot').textContent).toContain('Мои локации');
  });

  test('getRoot: возвращает элемент adminRoot', () => {
    // Техника тест-дизайна: классы эквивалентности
    expect(adminModule.getRoot()).toBe(document.getElementById('adminRoot'));
  });

  test('formatDateTime: для валидной даты возвращает строку', () => {
    // Техника тест-дизайна: классы эквивалентности
    const result = adminModule.formatDateTime('2026-05-01T15:00:00Z');
    expect(typeof result).toBe('string');
    expect(result.length).toBeGreaterThan(0);
  });

  test('renderNewRoomPage: рендерит форму создания помещения', async () => {
    // Техника тест-дизайна: сценарий использования
    await adminModule.renderNewRoomPage(12);

    expect(document.getElementById('adminRoot').textContent).toContain('Создание помещения');
    expect(document.getElementById('roomLocationId').value).toBe('12');
    expect(document.getElementById('saveDraftBtn')).not.toBeNull();
  });

  test('renderNewRoomPage: ошибка createAdminRoom показывает alert', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    window.Api = makeApi({
      createAdminRoom: jest.fn().mockRejectedValue(new Error('Ошибка создания помещения'))
    });

    await adminModule.renderNewRoomPage(1);

    document.getElementById('roomTitle').value = 'Зал';
    document.getElementById('roomDescription').value = 'Описание';
    document.getElementById('roomPrice').value = '1500';
    document.getElementById('roomCapacity').value = '8';

    document.getElementById('saveDraftBtn').click();
    await flushPromises();

    expect(global.alert).toHaveBeenCalledWith('Ошибка создания помещения');
    expect(console.error).toHaveBeenCalled();
  });

  test('renderRoomForm: в readonly режиме делает поля только для чтения', () => {
    // Техника тест-дизайна: классы эквивалентности
    const html = adminModule.renderRoomForm({
      location_id: 1,
      title: 'Зал',
      description: 'Описание',
      price: 1000,
      capacity: 5,
      available_from: '09:00',
      available_to: '21:00',
      images: ['/img/1.png']
    }, true);

    expect(html).toContain('readonly');
    expect(html).toContain('disabled');
  });

  test('renderRoomPage: rejected-помещение показывает причину отклонения', async () => {
    // Техника тест-дизайна: классы эквивалентности
    window.Api = makeApi({
      getAdminRoom: jest.fn().mockResolvedValue({
        id: 101,
        location_id: 1,
        title: 'Отклонённая комната',
        description: 'Описание',
        price: 1000,
        capacity: 10,
        available_from: '09:00',
        available_to: '21:00',
        images: ['/img/1.png'],
        status: 'rejected',
        rejection_reason: 'Недостаточно данных'
      })
    });

    await adminModule.renderRoomPage(101);

    expect(document.getElementById('adminRoot').textContent).toContain('Причина отклонения');
    expect(document.getElementById('adminRoot').textContent).toContain('Недостаточно данных');
    expect(document.getElementById('saveRoomBtn')).not.toBeNull();
    expect(document.getElementById('submitRoomBtn')).not.toBeNull();
  });

  test('renderRoomPage: archived-помещение не показывает кнопки сохранения', async () => {
    // Техника тест-дизайна: классы эквивалентности
    window.Api = makeApi({
      getAdminRoom: jest.fn().mockResolvedValue({
        id: 101,
        location_id: 1,
        title: 'Архивная комната',
        description: 'Описание',
        price: 1000,
        capacity: 10,
        available_from: '09:00',
        available_to: '21:00',
        images: ['/img/1.png'],
        status: 'archived',
        rejection_reason: '',
        archive: {}
      })
    });

    await adminModule.renderRoomPage(101);

    expect(document.getElementById('saveRoomBtn')).toBeNull();
    expect(document.getElementById('submitRoomBtn')).toBeNull();
    expect(document.getElementById('adminRoot').textContent).toContain('Помещение уже архивировано');
  });

  test('saveRoomChanges: при невалидной форме не вызывает updateAdminRoom', async () => {
    // Техника тест-дизайна: классы эквивалентности
    buildRoomFormDOM();
    document.getElementById('roomTitle').value = '';

    window.Api = makeApi({
      updateAdminRoom: jest.fn().mockResolvedValue({})
    });

    adminModule.__setCurrentRoomForTests?.({ id: 101, location_id: 1 });

    await adminModule.saveRoomChanges();

    expect(window.Api.updateAdminRoom).not.toHaveBeenCalled();
    expect(global.alert).toHaveBeenCalledWith('Введите название и описание');
  });

  test('saveRoomChanges: ошибка updateAdminRoom показывает alert', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    buildRoomFormDOM();

    window.Api = makeApi({
      updateAdminRoom: jest.fn().mockRejectedValue(new Error('Ошибка сохранения помещения'))
    });

    adminModule.__setCurrentRoomForTests?.({ id: 101, location_id: 1 });

    await adminModule.saveRoomChanges();

    expect(global.alert).toHaveBeenCalledWith('Ошибка сохранения помещения');
    expect(console.error).toHaveBeenCalled();
  });

  test('submitCurrentRoom: ошибка отправки на модерацию показывает alert', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    window.Api = makeApi({
      submitRoomForModeration: jest.fn().mockRejectedValue(new Error('Ошибка отправки на модерацию'))
    });

    adminModule.__setCurrentRoomForTests?.({ id: 101, location_id: 1 });

    await adminModule.submitCurrentRoom();

    expect(global.alert).toHaveBeenCalledWith('Ошибка отправки на модерацию');
    expect(console.error).toHaveBeenCalled();
  });

  test('loadBookingsForRoom: без статуса вызывает getAdminBookings только с room_id', async () => {
    // Техника тест-дизайна: классы эквивалентности
    document.getElementById('adminRoot').innerHTML = `
      <select id="bookingStatusFilter">
        <option value="">Все статусы</option>
      </select>
      <div id="bookingsTable"></div>
    `;

    window.Api = makeApi({
      getAdminBookings: jest.fn().mockResolvedValue({ items: [] })
    });

    await adminModule.loadBookingsForRoom(101);

    expect(window.Api.getAdminBookings).toHaveBeenCalledWith({
      room_id: 101
    });
    expect(document.getElementById('bookingsTable').textContent).toContain('Бронирований пока нет');
  });

  test('loadBookingsForRoom: со статусом вызывает getAdminBookings с room_id и status', async () => {
    // Техника тест-дизайна: таблица решений
    document.getElementById('adminRoot').innerHTML = `
      <select id="bookingStatusFilter">
        <option value="">Все статусы</option>
        <option value="booked">booked</option>
      </select>
      <div id="bookingsTable"></div>
    `;
    document.getElementById('bookingStatusFilter').value = 'booked';

    window.Api = makeApi({
      getAdminBookings: jest.fn().mockResolvedValue({ items: [] })
    });

    await adminModule.loadBookingsForRoom(101);

    expect(window.Api.getAdminBookings).toHaveBeenCalledWith({
      room_id: 101,
      status: 'booked'
    });
  });

  test('cancelBooking: при confirm=false отмена не выполняется', async () => {
    // Техника тест-дизайна: классы эквивалентности
    global.confirm.mockReturnValue(false);
    window.Api = makeApi({
      cancelAdminBooking: jest.fn().mockResolvedValue({})
    });

    adminModule.__setCurrentRoomForTests?.({ id: 101, location_id: 1 });

    await adminModule.cancelBooking('55');

    expect(window.Api.cancelAdminBooking).not.toHaveBeenCalled();
  });

  test('cancelBooking: ошибка отмены бронирования показывает alert', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    global.confirm.mockReturnValue(true);
    document.getElementById('adminRoot').innerHTML = `<div id="bookingsTable"></div>`;

    window.Api = makeApi({
      cancelAdminBooking: jest.fn().mockRejectedValue(new Error('Ошибка отмены бронирования'))
    });

    adminModule.__setCurrentRoomForTests?.({ id: 101, location_id: 1 });

    await adminModule.cancelBooking('55');

    expect(window.Api.cancelAdminBooking).toHaveBeenCalledWith('55');
    expect(global.alert).toHaveBeenCalledWith('Ошибка отмены бронирования');
    expect(console.error).toHaveBeenCalled();
  });

  test('bindArchiveButtons: immediate confirm=false не вызывает archiveRoom', async () => {
    // Техника тест-дизайна: классы эквивалентности
    document.getElementById('adminRoot').innerHTML = `
      <button id="archiveImmediateBtn">Архивировать сейчас</button>
      <button id="archiveScheduledBtn">Запланировать архивирование</button>
    `;
    global.confirm.mockReturnValue(false);

    const spy = jest.spyOn(adminModule, 'archiveRoom').mockResolvedValue();

    adminModule.bindArchiveButtons();
    document.getElementById('archiveImmediateBtn').click();

    expect(spy).not.toHaveBeenCalled();
  });

  test('bindArchiveButtons: scheduled confirm=false не вызывает archiveRoom', async () => {
    // Техника тест-дизайна: классы эквивалентности
    document.getElementById('adminRoot').innerHTML = `
      <button id="archiveImmediateBtn">Архивировать сейчас</button>
      <button id="archiveScheduledBtn">Запланировать архивирование</button>
    `;
    global.confirm.mockReturnValue(false);

    const spy = jest.spyOn(adminModule, 'archiveRoom').mockResolvedValue();

    adminModule.bindArchiveButtons();
    document.getElementById('archiveScheduledBtn').click();

    expect(spy).not.toHaveBeenCalled();
  });

  test('renderArchiveBlock: если booking_disabled/scheduled_for уже заданы, показывает pending-архивирование', () => {
    // Техника тест-дизайна: таблица решений
    const html = adminModule.renderArchiveBlock({
      status: 'published',
      archive: {
        booking_disabled: true,
        scheduled_for: '2026-05-01T15:00:00Z'
      }
    });

    expect(html).toContain('Помещение ожидает архивирования');
    expect(html).toContain('Новые бронирования отключены');
    expect(html).toContain('Запланировано на:');
  });

  test('bindLogout: если logoutBtn отсутствует, функция просто завершается без ошибки', () => {
    // Техника тест-дизайна: предугадывание ошибок
    document.getElementById('logoutBtn').remove();

    expect(() => adminModule.bindLogout()).not.toThrow();
  });

  test('renderAdminHome: при наличии локаций строит таблицу', async () => {
    // Техника тест-дизайна: сценарий использования
    window.Api = makeApi({
      getAdminLocations: jest.fn().mockResolvedValue({
        items: [
          {
            id: 1,
            company_name: 'Компания А',
            city: 'Москва',
            address: 'Адрес 1',
            rooms_count: 2,
            timezone: 'Europe/Moscow'
          },
          {
            id: 2,
            company_name: 'Компания Б',
            city: 'Казань',
            address: 'Адрес 2',
            rooms_count: 5,
            timezone: 'Europe/Moscow'
          }
        ]
      })
    });

    await adminModule.renderAdminHome();

    expect(document.querySelectorAll('#locationsTable tbody tr').length).toBe(2);
    expect(document.getElementById('adminRoot').textContent).toContain('Компания А');
    expect(document.getElementById('adminRoot').textContent).toContain('Компания Б');
  });
});