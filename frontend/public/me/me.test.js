
const meModule = require('./me.js');

describe('me.js', () => {
  async function flushPromises() {
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
  }

  function buildMeDOM() {
    document.body.innerHTML = `
      <main class="me-container">
        <section class="me-grid-wrap" aria-live="polite">
          <div class="me-grid">
            <div class="col" data-status="in_use">
              <div class="col-body" id="col-in_use"></div>
            </div>

            <div class="col" data-status="booked">
              <div class="col-body" id="col-booked"></div>
            </div>

            <div class="col" data-status="finished">
              <div class="col-body" id="col-finished"></div>
            </div>

            <div class="col" data-status="canceled">
              <div class="col-body" id="col-canceled"></div>
            </div>
          </div>

          <div id="emptyMessage" class="empty-message hidden">Ваша история бронирования пуста</div>
        </section>

        <footer id="meBottomBar">
          <div class="me-bottombar__inner">
            <div class="me-bottombar__center">
              <button id="logoutBtn" class="btn">Выйти</button>
            </div>
          </div>
        </footer>
      </main>

      <div id="confirmModal" class="modal-overlay hidden" aria-hidden="true">
        <div class="modal-content small">
          <div class="modal-body">
            <p id="confirmText">Уверены, что хотите отменить бронь?</p>
            <div class="modal-actions">
              <button id="confirmNo" class="btn">Нет</button>
              <button id="confirmYes" class="btn primary">Отменить</button>
            </div>
          </div>
        </div>
      </div>
    `;
  }

  function makeApi(overrides = {}) {
    return {
      getMe: jest.fn().mockResolvedValue({
        authenticated: true,
        user: { username: 'slava' }
      }),
      getMyBookings: jest.fn().mockResolvedValue({ items: [] }),
      cancelBooking: jest.fn().mockResolvedValue({ status: 'canceled' }),
      logoutUser: jest.fn().mockResolvedValue({}),
      ...overrides
    };
  }

  function sampleBooking(overrides = {}) {
    return {
      id: 'b1',
      room_id: '101',
      title: 'Переговорная',
      image_url: '/img/1.png',
      date: '2026-04-30',
      start_time: '10:00',
      end_time: '11:00',
      total_price: 1500,
      status: 'booked',
      ...overrides
    };
  }

  beforeEach(() => {
    jest.clearAllMocks();

    global.alert = jest.fn();

    global.console = {
      log: jest.fn(),
      warn: jest.fn(),
      error: jest.fn()
    };

    delete window.location;
    window.location = {
      href: ''
    };

    window.Api = makeApi();

    buildMeDOM();
    meModule.__resetStateForTests();
  });

  test('el: создаёт элемент и заполняет class, text и атрибуты', () => {
    // Техника тест-дизайна: классы эквивалентности
    const node = meModule.el('button', {
      class: 'btn primary',
      text: 'Отменить',
      type: 'button'
    });

    expect(node.tagName).toBe('BUTTON');
    expect(node.className).toBe('btn primary');
    expect(node.textContent).toBe('Отменить');
    expect(node.getAttribute('type')).toBe('button');
  });

  test('el: корректно устанавливает innerHTML через html', () => {
    // Техника тест-дизайна: классы эквивалентности
    const node = meModule.el('div', {
      html: '<span>test</span>'
    });

    expect(node.innerHTML).toBe('<span>test</span>');
  });

  test('initMePage: неавторизованный пользователь перенаправляется на /auth', async () => {
    // Техника тест-дизайна: классы эквивалентности
    window.Api = makeApi({
      getMe: jest.fn().mockResolvedValue({
        authenticated: false
      })
    });

    await meModule.initMePage();

    expect(window.location.href).toBe('/auth');
  });

  test('initMePage: для авторизованного пользователя вызывает loadBookings', async () => {
    // Техника тест-дизайна: сценарий использования
    window.Api = makeApi({
      getMe: jest.fn().mockResolvedValue({
        authenticated: true,
        user: { username: 'slava' }
      }),
      getMyBookings: jest.fn().mockResolvedValue({ items: [] })
    });

    await meModule.initMePage();

    expect(window.Api.getMe).toHaveBeenCalled();
    expect(window.Api.getMyBookings).toHaveBeenCalled();
  });

  test('initMePage: ошибка getMe показывает alert "Ошибка загрузки личного кабинета"', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    window.Api = makeApi({
      getMe: jest.fn().mockRejectedValue(new Error('boom'))
    });

    await meModule.initMePage();

    expect(global.alert).toHaveBeenCalledWith('Ошибка загрузки личного кабинета');
    expect(console.error).toHaveBeenCalled();
  });

  test('loadBookings: принимает массив из data.items и сохраняет в currentBookings', async () => {
    // Техника тест-дизайна: классы эквивалентности
    window.Api = makeApi({
      getMyBookings: jest.fn().mockResolvedValue({
        items: [sampleBooking()]
      })
    });

    await meModule.loadBookings();

    expect(meModule.__getCurrentBookingsForTests()).toEqual([sampleBooking()]);
  });

  test('loadBookings: принимает массив, если API вернул его напрямую', async () => {
    // Техника тест-дизайна: классы эквивалентности
    const items = [sampleBooking()];
    window.Api = makeApi({
      getMyBookings: jest.fn().mockResolvedValue(items)
    });

    await meModule.loadBookings();

    expect(meModule.__getCurrentBookingsForTests()).toEqual(items);
  });

  test('loadBookings: unauthorized при загрузке бронирований ведёт на /auth', async () => {
    // Техника тест-дизайна: таблица решений
    window.Api = makeApi({
      getMyBookings: jest.fn().mockRejectedValue({ code: 'unauthorized' })
    });

    await meModule.loadBookings();

    expect(window.location.href).toBe('/auth');
  });

  test('loadBookings: обычная ошибка загрузки показывает alert с message', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    window.Api = makeApi({
      getMyBookings: jest.fn().mockRejectedValue({ message: 'Не удалось загрузить список' })
    });

    await meModule.loadBookings();

    expect(global.alert).toHaveBeenCalledWith('Не удалось загрузить список');
    expect(console.error).toHaveBeenCalled();
  });

  test('clearColumns: очищает все четыре колонки', () => {
    // Техника тест-дизайна: сценарий использования
    document.getElementById('col-in_use').innerHTML = '<div>1</div>';
    document.getElementById('col-booked').innerHTML = '<div>2</div>';
    document.getElementById('col-finished').innerHTML = '<div>3</div>';
    document.getElementById('col-canceled').innerHTML = '<div>4</div>';

    meModule.clearColumns();

    expect(document.getElementById('col-in_use').innerHTML).toBe('');
    expect(document.getElementById('col-booked').innerHTML).toBe('');
    expect(document.getElementById('col-finished').innerHTML).toBe('');
    expect(document.getElementById('col-canceled').innerHTML).toBe('');
  });

  test('renderBookings: при пустом массиве показывает emptyMessage', () => {
    // Техника тест-дизайна: классы эквивалентности
    meModule.renderBookings([]);

    expect(document.getElementById('emptyMessage').classList.contains('hidden')).toBe(false);
  });

  test('renderBookings: распределяет карточки по колонкам по статусу', () => {
    // Техника тест-дизайна: сценарий использования
    const bookings = [
      sampleBooking({ id: '1', status: 'in_use' }),
      sampleBooking({ id: '2', status: 'booked' }),
      sampleBooking({ id: '3', status: 'finished' }),
      sampleBooking({ id: '4', status: 'canceled' })
    ];

    meModule.renderBookings(bookings);

    expect(document.querySelectorAll('#col-in_use .me-card').length).toBe(1);
    expect(document.querySelectorAll('#col-booked .me-card').length).toBe(1);
    expect(document.querySelectorAll('#col-finished .me-card').length).toBe(1);
    expect(document.querySelectorAll('#col-canceled .me-card').length).toBe(1);
    expect(document.getElementById('emptyMessage').classList.contains('hidden')).toBe(true);
  });

  test('renderBookings: неизвестный статус кладёт карточку в finished', () => {
    // Техника тест-дизайна: предугадывание ошибок
    meModule.renderBookings([
      sampleBooking({ id: 'x', status: 'unknown_status' })
    ]);

    expect(document.querySelectorAll('#col-finished .me-card').length).toBe(1);
  });

  test('createCard: создаёт карточку и добавляет кнопку отмены только для status=booked', () => {
    // Техника тест-дизайна: таблица решений
    const bookedCard = meModule.createCard(sampleBooking({ status: 'booked' }));
    const finishedCard = meModule.createCard(sampleBooking({ id: '2', status: 'finished' }));

    expect(bookedCard.querySelector('.cancel-btn')).not.toBeNull();
    expect(finishedCard.querySelector('.cancel-btn')).toBeNull();
  });

  test('createCard: при отсутствии image_url подставляет placeholder', () => {
    // Техника тест-дизайна: классы эквивалентности
    const card = meModule.createCard(sampleBooking({ image_url: '' }));

    expect(card.querySelector('img').src).toContain('/shared/placeholders/room-placeholder.svg');
  });

  test('createCard: клик по карточке ведёт на страницу комнаты', () => {
    // Техника тест-дизайна: сценарий использования
    const card = meModule.createCard(sampleBooking({ room_id: '777' }));

    card.click();

    expect(window.location.href).toBe('/room/777');
  });

  test('createCard: клик по кнопке отмены не открывает страницу комнаты', () => {
    // Техника тест-дизайна: сценарий использования
    const card = meModule.createCard(sampleBooking({ room_id: '777' }));
    const cancelBtn = card.querySelector('.cancel-btn');

    cancelBtn.click();

    expect(window.location.href).toBe('');
  });

  test('openConfirmCancel: открывает модалку и сохраняет pendingCancelBookingId', () => {
    // Техника тест-дизайна: переходы состояний
    meModule.openConfirmCancel('b55');

    expect(meModule.__getPendingCancelBookingIdForTests()).toBe('b55');
    expect(document.getElementById('confirmModal').classList.contains('hidden')).toBe(false);
    expect(document.getElementById('confirmModal').getAttribute('aria-hidden')).toBe('false');
    expect(document.getElementById('confirmText').textContent)
      .toBe('Уверены, что хотите отменить бронь?');
  });

  test('closeConfirmCancel: закрывает модалку и сбрасывает pendingCancelBookingId', () => {
    // Техника тест-дизайна: переходы состояний
    meModule.__setPendingCancelBookingIdForTests('b77');
    document.getElementById('confirmModal').classList.remove('hidden');
    document.getElementById('confirmModal').setAttribute('aria-hidden', 'false');

    meModule.closeConfirmCancel();

    expect(meModule.__getPendingCancelBookingIdForTests()).toBeNull();
    expect(document.getElementById('confirmModal').classList.contains('hidden')).toBe(true);
    expect(document.getElementById('confirmModal').getAttribute('aria-hidden')).toBe('true');
  });

  test('cancelBooking: без bookingId ничего не делает', async () => {
    // Техника тест-дизайна: граничные условия
    await meModule.cancelBooking('');

    expect(window.Api.cancelBooking).not.toHaveBeenCalled();
  });

  test('cancelBooking: успешная отмена меняет статус брони и перезагружает список', async () => {
    // Техника тест-дизайна: сценарий использования
    meModule.__setCurrentBookingsForTests([
      sampleBooking({ id: 'b1', status: 'booked' }),
      sampleBooking({ id: 'b2', status: 'finished' })
    ]);

    window.Api = makeApi({
      cancelBooking: jest.fn().mockResolvedValue({ status: 'canceled' }),
      getMyBookings: jest.fn().mockResolvedValue({
        items: [
          sampleBooking({ id: 'b1', status: 'canceled' }),
          sampleBooking({ id: 'b2', status: 'finished' })
        ]
      })
    });

    await meModule.cancelBooking('b1');

    expect(window.Api.cancelBooking).toHaveBeenCalledWith('b1');
    expect(window.Api.getMyBookings).toHaveBeenCalled();
    expect(meModule.__getCurrentBookingsForTests()[0].status).toBe('canceled');
  });

  test('cancelBooking: unauthorized показывает alert и ведёт на /auth', async () => {
    // Техника тест-дизайна: таблица решений
    window.Api = makeApi({
      cancelBooking: jest.fn().mockRejectedValue({ code: 'unauthorized' })
    });

    await meModule.cancelBooking('b1');

    expect(global.alert).toHaveBeenCalledWith('Необходимо авторизоваться');
    expect(window.location.href).toBe('/auth');
  });

  test('cancelBooking: cannot_cancel_booking показывает alert и перезагружает список', async () => {
    // Техника тест-дизайна: таблица решений
    window.Api = makeApi({
      cancelBooking: jest.fn().mockRejectedValue({
        code: 'cannot_cancel_booking',
        message: 'Бронь уже используется'
      }),
      getMyBookings: jest.fn().mockResolvedValue({ items: [] })
    });

    await meModule.cancelBooking('b1');

    expect(global.alert).toHaveBeenCalledWith('Бронь уже используется');
    expect(window.Api.getMyBookings).toHaveBeenCalled();
  });

  test('cancelBooking: неизвестная ошибка показывает alert', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    window.Api = makeApi({
      cancelBooking: jest.fn().mockRejectedValue({
        message: 'Ошибка отмены бронирования'
      })
    });

    await meModule.cancelBooking('b1');

    expect(global.alert).toHaveBeenCalledWith('Ошибка отмены бронирования');
    expect(console.error).toHaveBeenCalled();
  });

  test('bindModalEvents: confirmNo закрывает модалку', () => {
    // Техника тест-дизайна: переходы состояний
    meModule.openConfirmCancel('b1');
    meModule.bindModalEvents();

    document.getElementById('confirmNo').click();

    expect(document.getElementById('confirmModal').classList.contains('hidden')).toBe(true);
    expect(meModule.__getPendingCancelBookingIdForTests()).toBeNull();
  });

  test('bindModalEvents: клик по overlay закрывает модалку', () => {
    // Техника тест-дизайна: сценарий использования
    meModule.openConfirmCancel('b1');
    meModule.bindModalEvents();

    document.getElementById('confirmModal')
      .dispatchEvent(new MouseEvent('click', { bubbles: true }));

    expect(document.getElementById('confirmModal').classList.contains('hidden')).toBe(true);
  });

  test('bindModalEvents: Escape закрывает модалку', () => {
    // Техника тест-дизайна: переходы состояний
    meModule.openConfirmCancel('b1');
    meModule.bindModalEvents();

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));

    expect(document.getElementById('confirmModal').classList.contains('hidden')).toBe(true);
    expect(meModule.__getPendingCancelBookingIdForTests()).toBeNull();
  });


  test('bindLogout: успешный logout ведёт на главную', async () => {
    // Техника тест-дизайна: сценарий использования
    meModule.bindLogout();

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

    meModule.bindLogout();

    document.getElementById('logoutBtn').click();
    await flushPromises();

    expect(global.alert).toHaveBeenCalledWith('Ошибка выхода');
    expect(console.error).toHaveBeenCalled();
  });

  test('bindLogout: если logoutBtn отсутствует, функция завершается без ошибки', () => {
    // Техника тест-дизайна: предугадывание ошибок
    document.getElementById('logoutBtn').remove();

    expect(() => meModule.bindLogout()).not.toThrow();
  });
});