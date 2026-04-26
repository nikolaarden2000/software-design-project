/**
 * home.test.js
 * Тесты для home.js
 */

async function flushPromises() {
  await Promise.resolve();
  await Promise.resolve();
  await Promise.resolve();
  await Promise.resolve();
}

describe('home.js', () => {
  let home;
  let wsInstances;

  class MockWebSocket {
    static CONNECTING = 0;
    static OPEN = 1;
    static CLOSING = 2;
    static CLOSED = 3;

    constructor(url) {
      this.url = url;
      this.readyState = MockWebSocket.CONNECTING;
      this.sent = [];
      this.listeners = {};
      this.close = jest.fn(() => {
        this.readyState = MockWebSocket.CLOSED;
      });
      this.send = jest.fn((data) => {
        this.sent.push(data);
      });
      wsInstances.push(this);
    }

    addEventListener(type, cb) {
      if (!this.listeners[type]) this.listeners[type] = [];
      this.listeners[type].push(cb);
    }

    emit(type, payload = {}) {
      const list = this.listeners[type] || [];
      list.forEach(fn => fn(payload));
    }
  }

  class MockIntersectionObserver {
    constructor(cb) {
      this.cb = cb;
      this.observe = jest.fn();
      this.disconnect = jest.fn();
    }
  }

  function buildDOM() {
    document.body.innerHTML = `
      <div id="cityName"></div>
      <button id="cityBtn"></button>
      <dialog id="cityModal"></dialog>
      <ul id="cityList">
        <li class="city-item">Москва</li>
        <li class="city-item">Казань</li>
        <li class="city-item">Екатеринбург</li>
      </ul>
      <input id="citySearch" />
      <button id="cityOk"></button>
      <button id="cityCancel"></button>

      <button id="authButton" data-auth="0"></button>
      <a id="brand" href="/">Brand</a>

      <div id="cardsWrapper"></div>
      <div id="statusBar" hidden></div>

      <div id="companyToggleWrap"></div>
      <div id="companyList"></div>

      <input id="priceInput" />
      <input id="capacityInput" />
      <button id="applyFilters"></button>
      <button id="clearFilters"></button>
    `;
    document.body.dataset.initialCity = 'Москва';

    const cityModal = document.getElementById('cityModal');
    cityModal.showModal = jest.fn();
    cityModal.close = jest.fn();
    cityModal.getBoundingClientRect = jest.fn(() => ({
      left: 10, right: 100, top: 10, bottom: 100
    }));
  }

  function loadModule() {
    jest.isolateModules(() => {
      home = require('./home.js');
    });
    jest.clearAllMocks();
  }

  beforeEach(() => {
    jest.resetModules();
    jest.useFakeTimers();

    wsInstances = [];

    document.body.innerHTML = '';
    buildDOM();

    global.console = {
      log: jest.fn(),
      warn: jest.fn(),
      error: jest.fn()
    };

    global.IntersectionObserver = MockIntersectionObserver;
    global.WebSocket = MockWebSocket;
    global.WebSocket.OPEN = MockWebSocket.OPEN;
    global.WebSocket.CONNECTING = MockWebSocket.CONNECTING;
    global.WebSocket.CLOSED = MockWebSocket.CLOSED;
    global.WebSocket.CLOSING = MockWebSocket.CLOSING;

    delete window.location;
    window.location = {
      protocol: 'http:',
      host: 'localhost:8080',
      href: ''
    };

    loadModule();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
    jest.restoreAllMocks();
  });

  test('getWsUrl: возвращает ws URL для http протокола', () => {
    // Техника тест-дизайна: классы эквивалентности
    window.location.protocol = 'http:';
    window.location.host = 'example.com';

    expect(home.getWsUrl()).toBe('ws://example.com/ws/home');
  });

  test('getWsUrl: возвращает wss URL для https протокола', () => {
    // Техника тест-дизайна: классы эквивалентности
    window.location.protocol = 'https:';
    window.location.host = 'secure.example.com';

    expect(home.getWsUrl()).toBe('wss://secure.example.com/ws/home');
  });

  test('navigate: меняет window.location.href на переданный URL', () => {
    // Техника тест-дизайна: классы эквивалентности
    home.navigate('/auth');

    expect(window.location.href).toBe('/auth');
  });

  test('setFiltersEnabled: включает кнопки и чекбоксы фильтров', () => {
    // Техника тест-дизайна: классы эквивалентности
    const companyList = document.getElementById('companyList');
    companyList.innerHTML = `
      <input type="checkbox" />
      <input type="checkbox" />
    `;

    home.setFiltersEnabled(true);

    expect(document.getElementById('applyFilters').disabled).toBe(false);
    expect(document.getElementById('clearFilters').disabled).toBe(false);
    companyList.querySelectorAll('input[type="checkbox"]').forEach(cb => {
      expect(cb.disabled).toBe(false);
    });
  });

  test('setFiltersEnabled: выключает кнопки и чекбоксы фильтров', () => {
    // Техника тест-дизайна: классы эквивалентности
    const companyList = document.getElementById('companyList');
    companyList.innerHTML = `
      <input type="checkbox" />
      <input type="checkbox" />
    `;

    home.setFiltersEnabled(false);

    expect(document.getElementById('applyFilters').disabled).toBe(true);
    expect(document.getElementById('clearFilters').disabled).toBe(true);
    companyList.querySelectorAll('input[type="checkbox"]').forEach(cb => {
      expect(cb.disabled).toBe(true);
    });
  });

  test('filterCityList: скрывает неподходящие города и оставляет подходящие', () => {
    // Техника тест-дизайна: классы эквивалентности
    home.filterCityList('каз');

    const items = Array.from(document.querySelectorAll('.city-item'));
    const moscow = items.find(i => i.textContent.trim() === 'Москва');
    const kazan = items.find(i => i.textContent.trim() === 'Казань');

    expect(moscow.style.display).toBe('none');
    expect(kazan.style.display).toBe('block');
  });

  test('filterCityList: при отсутствии совпадений добавляет сообщение "Город не найден"', () => {
    // Техника тест-дизайна: классы эквивалентности
    home.filterCityList('Тверь');

    const items = Array.from(document.querySelectorAll('.city-item'));
    expect(items.some(i => i.textContent === 'Город не найден')).toBe(true);
  });

  test('showCityError: отображает сообщение об ошибке и hideCityError удаляет его', () => {
    // Техника тест-дизайна: сценарий использования
    home.showCityError('Такого города не существует');

    let errorEl = document.getElementById('cityError');
    expect(errorEl).not.toBeNull();
    expect(errorEl.textContent).toBe('Такого города не существует');

    home.hideCityError();

    errorEl = document.getElementById('cityError');
    expect(errorEl).toBeNull();
  });

  test('showCityError: автоматически скрывает ошибку через 3000 мс', () => {
    // Техника тест-дизайна: граничные условия
    home.showCityError('Ошибка города');

    expect(document.getElementById('cityError')).not.toBeNull();

    jest.advanceTimersByTime(2999);
    expect(document.getElementById('cityError')).not.toBeNull();

    jest.advanceTimersByTime(1);
    expect(document.getElementById('cityError')).toBeNull();
  });

  test('setCity: несуществующий город показывает ошибку и не меняет cityName', () => {
    // Техника тест-дизайна: классы эквивалентности
    document.getElementById('cityName').textContent = 'Москва';

    home.setCity('Тверь');

    expect(document.getElementById('cityName').textContent).toBe('Москва');
    expect(document.getElementById('cityError')).not.toBeNull();
    expect(document.getElementById('cityError').textContent).toBe('Такого города не существует');
  });

  test('setCity: валидный новый город обновляет cityName и dataset', () => {
    // Техника тест-дизайна: классы эквивалентности
    home.setCity('Казань');

    expect(document.getElementById('cityName').textContent).toBe('Казань');
    expect(document.body.dataset.initialCity).toBe('Казань');
  });

  test('setCity: повторная установка того же города ничего не меняет', () => {
    // Техника тест-дизайна: классы эквивалентности
    document.getElementById('cityName').textContent = 'Москва';
    const initialWsCount = wsInstances.length;

    home.setCity('Москва');

    expect(document.getElementById('cityName').textContent).toBe('Москва');
    expect(wsInstances.length).toBe(initialWsCount);
  });

  test('renderCards: при пустом массиве показывает сообщение об отсутствии результатов', () => {
    // Техника тест-дизайна: классы эквивалентности
    home.renderCards([], false);

    const cardsWrapper = document.getElementById('cardsWrapper');
    expect(cardsWrapper.textContent).toContain('Ничего не найдено по выбранным фильтрам');
  });

  test('renderCards: создаёт карточку с данными помещения', () => {
    // Техника тест-дизайна: сценарий использования
    home.renderCards([
      {
        id: 'room-1',
        title: 'Переговорная',
        company: 'Компания А',
        address: 'Улица 1',
        capacity: 12,
        price: 1500,
        image: 'img.png'
      }
    ], false);

    const card = document.querySelector('.card');
    expect(card).not.toBeNull();
    expect(card.getAttribute('aria-label')).toBe('Переговорная — Компания А');
    expect(document.querySelector('.card__title').textContent).toBe('Переговорная');
    expect(document.querySelector('.card__company').textContent).toBe('Компания А');
    expect(document.querySelector('.card__address').textContent).toBe('Улица 1');
    expect(document.querySelector('.price-badge').textContent).toBe('1500 ₽/ч');
  });

  test('renderCards: клик по карточке с id выполняет переход на страницу комнаты', () => {
    // Техника тест-дизайна: сценарий использования
    home.renderCards([
      {
        id: 'room-42',
        title: 'Зал',
        company: 'Компания B',
        address: 'Адрес',
        capacity: 5,
        price: 900
      }
    ], false);

    document.querySelector('.card').click();

    expect(window.location.href).toBe('/room/room-42');
  });

  test('renderCards: карточка без id не вызывает переход', () => {
    // Техника тест-дизайна: предугадывание ошибок
    home.renderCards([
      {
        title: 'Без ID',
        company: 'Компания B',
        address: 'Адрес',
        capacity: 5,
        price: 900
      }
    ], false);

    document.querySelector('.card').click();

    expect(window.location.href).toBe('');
    expect(console.warn).toHaveBeenCalled();
  });

  test('showCenteredMessage: отображает сообщение и кнопку повтора при showRetry=true', () => {
    // Техника тест-дизайна: классы эквивалентности
    home.showCenteredMessage('Ошибка подключения', true);

    const wrapper = document.getElementById('cardsWrapper');
    expect(wrapper.textContent).toContain('Ошибка подключения');
    expect(wrapper.textContent).toContain('Попробовать сейчас');
  });

  test('refreshFilterValues: при пустом списке компаний показывает "Нет данных"', () => {
    // Техника тест-дизайна: классы эквивалентности
    home.refreshFilterValues();

    expect(document.getElementById('companyList').innerHTML).toContain('Нет данных');
  });

  test('handleWsMessage: первый массив компаний создаёт чекбоксы фильтров', () => {
    // Техника тест-дизайна: сценарий использования
    const openWs = wsInstances[0];
    openWs.readyState = MockWebSocket.OPEN;

    home.handleWsMessage(JSON.stringify(['Компания А', 'Компания Б']));

    const checkboxes = document.querySelectorAll('#companyList input[type="checkbox"]');
    expect(checkboxes.length).toBe(2);
    expect(checkboxes[0].value).toBe('Компания А');
    expect(checkboxes[1].value).toBe('Компания Б');
    expect(openWs.send).toHaveBeenCalledWith('100');
  });

  test('handleWsMessage: null от сервера показывает сообщение об отсутствии помещений', () => {
    // Техника тест-дизайна: таблица решений
    const openWs = wsInstances[0];
    openWs.readyState = MockWebSocket.OPEN;

    home.handleWsMessage('null');

    expect(document.getElementById('cardsWrapper').textContent)
      .toContain('В этом городе нет помещений для бронирования');
    expect(openWs.close).toHaveBeenCalled();
  });

  test('handleWsMessage: пустой массив после компаний показывает сообщение об отсутствии помещений', () => {
    // Техника тест-дизайна: таблица решений
    const openWs = wsInstances[0];
    openWs.readyState = MockWebSocket.OPEN;

    home.handleWsMessage(JSON.stringify(['Компания А']));
    home.handleWsMessage(JSON.stringify([]));

    expect(document.getElementById('cardsWrapper').textContent)
      .toContain('В этом городе нет помещений для бронирования');
    expect(openWs.close).toHaveBeenCalled();
  });

  test('handleWsMessage: массив карточек добавляет элементы на страницу', () => {
    // Техника тест-дизайна: сценарий использования
    const openWs = wsInstances[0];
    openWs.readyState = MockWebSocket.OPEN;

    home.handleWsMessage(JSON.stringify(['Компания А']));
    home.handleWsMessage(JSON.stringify([
      {
        id: '1',
        title: 'Комната 1',
        company: 'Компания А',
        address: 'Адрес 1',
        capacity: 10,
        price: 1000
      }
    ]));

    expect(document.querySelectorAll('.card').length).toBe(1);
    expect(document.getElementById('cardsWrapper').textContent).toContain('Комната 1');
  });

  test('requestMoreData: при открытом websocket отправляет размер батча', () => {
    // Техника тест-дизайна: классы эквивалентности
    const openWs = wsInstances[0];
    openWs.readyState = MockWebSocket.OPEN;

    home.requestMoreData();

    expect(openWs.send).toHaveBeenCalledWith('100');
    expect(document.getElementById('statusBar').textContent).toBe('Запрашиваем данные...');
  });

  test('requestMoreData: при закрытом websocket не падает с ошибкой', () => {
    // Техника тест-дизайна: предугадывание ошибок
    const closedWs = wsInstances[0];
    closedWs.readyState = MockWebSocket.CLOSED;

    expect(() => home.requestMoreData()).not.toThrow();
  });

  test('applyFilters: по цене оставляет только подходящие карточки', () => {
    // Техника тест-дизайна: классы эквивалентности
    const openWs = wsInstances[0];
    openWs.readyState = MockWebSocket.OPEN;

    home.handleWsMessage(JSON.stringify(['Компания А', 'Компания Б']));
    home.handleWsMessage(JSON.stringify([
      {
        id: '1',
        title: 'Комната 1',
        company: 'Компания А',
        address: 'Адрес 1',
        capacity: 2,
        price: 1000
      },
      {
        id: '2',
        title: 'Комната 2',
        company: 'Компания Б',
        address: 'Адрес 2',
        capacity: 6,
        price: 3000
      }
    ]));

    document.getElementById('priceInput').value = '1500';
    home.applyFilters(true);

    const cards = document.querySelectorAll('.card');
    expect(cards.length).toBe(1);
    expect(document.getElementById('cardsWrapper').textContent).toContain('Комната 1');
    expect(document.getElementById('cardsWrapper').textContent).not.toContain('Комната 2');
  });

  test('applyFilters: по вместимости оставляет только подходящие карточки', () => {
    // Техника тест-дизайна: классы эквивалентности
    const openWs = wsInstances[0];
    openWs.readyState = MockWebSocket.OPEN;

    home.handleWsMessage(JSON.stringify(['Компания А', 'Компания Б']));
    home.handleWsMessage(JSON.stringify([
      {
        id: '1',
        title: 'Комната 1',
        company: 'Компания А',
        address: 'Адрес 1',
        capacity: 2,
        price: 1000
      },
      {
        id: '2',
        title: 'Комната 2',
        company: 'Компания Б',
        address: 'Адрес 2',
        capacity: 8,
        price: 1500
      }
    ]));

    document.getElementById('capacityInput').value = '5';
    home.applyFilters(true);

    const cards = document.querySelectorAll('.card');
    expect(cards.length).toBe(1);
    expect(document.getElementById('cardsWrapper').textContent).toContain('Комната 2');
  });

  test('applyFilters: по выбранной компании оставляет только её карточки', () => {
    // Техника тест-дизайна: классы эквивалентности
    const openWs = wsInstances[0];
    openWs.readyState = MockWebSocket.OPEN;

    home.handleWsMessage(JSON.stringify(['Компания А', 'Компания Б']));
    home.handleWsMessage(JSON.stringify([
      {
        id: '1',
        title: 'Комната 1',
        company: 'Компания А',
        address: 'Адрес 1',
        capacity: 2,
        price: 1000
      },
      {
        id: '2',
        title: 'Комната 2',
        company: 'Компания Б',
        address: 'Адрес 2',
        capacity: 8,
        price: 1500
      }
    ]));

    const checkboxes = document.querySelectorAll('#companyList input[type="checkbox"]');
    checkboxes[1].checked = true;

    home.applyFilters(true);

    const cards = document.querySelectorAll('.card');
    expect(cards.length).toBe(1);
    expect(document.getElementById('cardsWrapper').textContent).toContain('Комната 2');
    expect(document.getElementById('cardsWrapper').textContent).not.toContain('Комната 1');
  });

  test('applyFilters: комбинация цена + вместимость + компания оставляет одну подходящую карточку', () => {
    // Техника тест-дизайна: попарное тестирование
    const openWs = wsInstances[0];
    openWs.readyState = MockWebSocket.OPEN;

    home.handleWsMessage(JSON.stringify(['Компания А', 'Компания Б']));
    home.handleWsMessage(JSON.stringify([
      {
        id: '1',
        title: 'Комната 1',
        company: 'Компания А',
        address: 'Адрес 1',
        capacity: 2,
        price: 1000
      },
      {
        id: '2',
        title: 'Комната 2',
        company: 'Компания Б',
        address: 'Адрес 2',
        capacity: 8,
        price: 1400
      },
      {
        id: '3',
        title: 'Комната 3',
        company: 'Компания Б',
        address: 'Адрес 3',
        capacity: 3,
        price: 5000
      }
    ]));

    document.getElementById('priceInput').value = '1500';
    document.getElementById('capacityInput').value = '5';
    const checkboxes = document.querySelectorAll('#companyList input[type="checkbox"]');
    checkboxes[1].checked = true;

    home.applyFilters(true);

    const cards = document.querySelectorAll('.card');
    expect(cards.length).toBe(1);
    expect(document.getElementById('cardsWrapper').textContent).toContain('Комната 2');
  });

  test('applyFilters: при пустом allItems и userTriggered=true показывает пустой результат', () => {
    // Техника тест-дизайна: классы эквивалентности
    home.resetState();
    home.applyFilters(true);

    expect(document.getElementById('cardsWrapper').textContent)
      .toContain('Ничего не найдено по выбранным фильтрам');
  });

  test('refreshFilterValues: при количестве компаний больше 6 создаёт кнопку "Показать всё"', () => {
    // Техника тест-дизайна: граничные условия
    const openWs = wsInstances[0];
    openWs.readyState = MockWebSocket.OPEN;

    home.handleWsMessage(JSON.stringify([
      'A', 'B', 'C', 'D', 'E', 'F', 'G'
    ]));

    expect(document.getElementById('companyToggleWrap').textContent).toContain('Показать всё');
  });

  test('refreshFilterValues: при количестве компаний 6 и меньше не создаёт кнопку раскрытия', () => {
    // Техника тест-дизайна: граничные условия
    const openWs = wsInstances[0];
    openWs.readyState = MockWebSocket.OPEN;

    home.handleWsMessage(JSON.stringify([
      'A', 'B', 'C', 'D', 'E', 'F'
    ]));

    expect(document.getElementById('companyToggleWrap').textContent).toBe('');
  });

  test('connectWebSocket: создаёт websocket с корректным URL', () => {
    // Техника тест-дизайна: сценарий использования
    home.resetState();
    home.connectWebSocket();

    expect(wsInstances.length).toBeGreaterThan(0);
    expect(wsInstances[wsInstances.length - 1].url).toBe('ws://localhost:8080/ws/home');
  });

  test('connectWebSocket: при событии open отправляет текущий город', () => {
    // Техника тест-дизайна: сценарий использования
    home.resetState();
    home.connectWebSocket();

    const ws = wsInstances[wsInstances.length - 1];
    ws.readyState = MockWebSocket.OPEN;
    ws.emit('open');

    expect(ws.send).toHaveBeenCalledWith('Москва');
    expect(document.getElementById('statusBar').textContent)
      .toBe('Соединение установлено. Проверяю наличие помещений...');
  });

  test('scheduleReconnect: планирует повторное подключение через 1 секунду', () => {
    // Техника тест-дизайна: граничные условия
    const spyConnect = jest.spyOn(home, 'connectWebSocket').mockImplementation(() => {});

    home.resetState();
    home.scheduleReconnect();

    expect(document.getElementById('cardsWrapper').textContent)
      .toContain('Не удалось подключиться к серверу. Попытка через 1с...');

    jest.advanceTimersByTime(1000);

    expect(spyConnect).toHaveBeenCalled();
  });

  test('brand link: клик по бренду ведёт на главную страницу', () => {
    // Техника тест-дизайна: сценарий использования
    const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true });
    document.getElementById('brand').dispatchEvent(clickEvent);

    expect(window.location.href).toBe('/');
  });

  test('auth button: для неавторизованного пользователя ведёт на /auth', () => {
    // Техника тест-дизайна: таблица решений
    const btn = document.getElementById('authButton');
    btn.dataset.auth = '0';

    btn.click();

    expect(window.location.href).toBe('/auth');
  });

  test('auth button: для авторизованного пользователя ведёт на /me', () => {
    // Техника тест-дизайна: таблица решений
    const btn = document.getElementById('authButton');
    btn.dataset.auth = '1';

    btn.click();

    expect(window.location.href).toBe('/me');
  });
    test('handleWsMessage: некорректный JSON не выбрасывает исключение и пишет warning', () => {
    // Техника тест-дизайна: предугадывание ошибок
    expect(() => home.handleWsMessage('not-json')).not.toThrow();
    expect(console.warn).toHaveBeenCalled();
  });

  test('requestMoreData: при активной фильтрации не отправляет новый запрос', () => {
    // Техника тест-дизайна: классы эквивалентности
    const openWs = wsInstances[0];
    openWs.readyState = MockWebSocket.OPEN;

    home.handleWsMessage(JSON.stringify(['Компания А']));
    home.handleWsMessage(JSON.stringify([
      {
        id: '1',
        title: 'Комната 1',
        company: 'Компания А',
        address: 'Адрес 1',
        capacity: 2,
        price: 1000
      }
    ]));

    document.getElementById('priceInput').value = '1500';
    home.applyFilters(true); // делает filtering = true при активном фильтре

    openWs.send.mockClear();
    home.requestMoreData();

    expect(openWs.send).not.toHaveBeenCalled();
  });

  test('handleWsMessage: если список компаний пришёл при закрытом websocket, увеличивает pendingRequestCount', () => {
    // Техника тест-дизайна: таблица решений
    const ws = wsInstances[0];
    ws.readyState = MockWebSocket.CLOSED;

    home.handleWsMessage(JSON.stringify(['Компания А', 'Компания Б']));

    expect(home.resetState).toBeDefined(); // просто чтобы использовать exported module
    expect(window.debug_state().pendingRequestCount).toBe(100);
  });

  test('connectWebSocket: при open отправляет накопленный pendingRequestCount и обнуляет его', () => {
    // Техника тест-дизайна: сценарий использования
    const firstWs = wsInstances[0];
    firstWs.readyState = MockWebSocket.CLOSED;

    home.handleWsMessage(JSON.stringify(['Компания А']));
    expect(window.debug_state().pendingRequestCount).toBe(100);

    home.connectWebSocket();

    const secondWs = wsInstances[wsInstances.length - 1];
    secondWs.readyState = MockWebSocket.OPEN;
    secondWs.emit('open');

    expect(secondWs.send).toHaveBeenCalledWith('Москва');
    expect(secondWs.send).toHaveBeenCalledWith('100');
    expect(window.debug_state().pendingRequestCount).toBe(0);
  });

  test('showCenteredMessage: кнопка повтора вызывает повторное подключение', () => {
    // Техника тест-дизайна: сценарий использования
    const spyConnect = jest.spyOn(home, 'connectWebSocket').mockImplementation(() => {});

    home.showCenteredMessage('Ошибка подключения', true);

    const retryBtn = Array.from(document.querySelectorAll('button'))
      .find(btn => btn.textContent === 'Попробовать сейчас');

    expect(retryBtn).toBeDefined();

    retryBtn.click();

    expect(document.getElementById('cardsWrapper').textContent).toContain('Подключаемся...');
    expect(spyConnect).toHaveBeenCalled();
  });

  test('priceInput: input удаляет нецифровые символы и ведущий ноль', () => {
    // Техника тест-дизайна: классы эквивалентности
    const priceInput = document.getElementById('priceInput');

    priceInput.value = '0a12b';
    priceInput.dispatchEvent(new Event('input', { bubbles: true }));

    expect(priceInput.value).toBe('12');
  });

  test('priceInput: blur очищает значение, если после нормализации цена равна 0', () => {
    // Техника тест-дизайна: граничные условия
    const priceInput = document.getElementById('priceInput');

    priceInput.value = '0';
    priceInput.dispatchEvent(new Event('blur', { bubbles: true }));

    expect(priceInput.value).toBe('');
  });

  test('priceInput: keydown для недопустимого символа вызывает preventDefault', () => {
    // Техника тест-дизайна: классы эквивалентности
    const priceInput = document.getElementById('priceInput');
    const event = new KeyboardEvent('keydown', { key: 'a', bubbles: true });
    const preventDefault = jest.spyOn(event, 'preventDefault');

    priceInput.dispatchEvent(event);

    expect(preventDefault).toHaveBeenCalled();
  });

  test('capacityInput: input оставляет только цифры', () => {
    // Техника тест-дизайна: классы эквивалентности
    const capacityInput = document.getElementById('capacityInput');

    capacityInput.value = '1a2b3';
    capacityInput.dispatchEvent(new Event('input', { bubbles: true }));

    expect(capacityInput.value).toBe('123');
  });

  test('clearFilters button: очищает поля и снимает чекбоксы', () => {
    // Техника тест-дизайна: сценарий использования
    const openWs = wsInstances[0];
    openWs.readyState = MockWebSocket.OPEN;

    home.handleWsMessage(JSON.stringify(['Компания А', 'Компания Б']));
    home.handleWsMessage(JSON.stringify([
      {
        id: '1',
        title: 'Комната 1',
        company: 'Компания А',
        address: 'Адрес 1',
        capacity: 2,
        price: 1000
      },
      {
        id: '2',
        title: 'Комната 2',
        company: 'Компания Б',
        address: 'Адрес 2',
        capacity: 8,
        price: 1500
      }
    ]));

    document.getElementById('priceInput').value = '1500';
    document.getElementById('capacityInput').value = '5';
    const checkboxes = document.querySelectorAll('#companyList input[type="checkbox"]');
    checkboxes[0].checked = true;
    checkboxes[1].checked = true;

    document.getElementById('clearFilters').click();

    expect(document.getElementById('priceInput').value).toBe('');
    expect(document.getElementById('capacityInput').value).toBe('');
    checkboxes.forEach(cb => expect(cb.checked).toBe(false));
  });

  test('cityOk: без выбора и без ввода показывает ошибку', () => {
    // Техника тест-дизайна: классы эквивалентности
    document.getElementById('citySearch').value = '';

    document.getElementById('cityOk').click();

    expect(document.getElementById('cityError')).not.toBeNull();
    expect(document.getElementById('cityError').textContent)
      .toBe('Выберите город из списка или введите название');
  });

  test('citySearch Enter: пустое значение показывает ошибку', () => {
    // Техника тест-дизайна: классы эквивалентности
    const citySearch = document.getElementById('citySearch');
    citySearch.value = '';

    citySearch.dispatchEvent(new KeyboardEvent('keyup', { key: 'Enter', bubbles: true }));

    expect(document.getElementById('cityError')).not.toBeNull();
    expect(document.getElementById('cityError').textContent).toBe('Введите название города');
  });

  test('cityList click: выбирает город и записывает его в поле поиска', () => {
    // Техника тест-дизайна: сценарий использования
    const item = Array.from(document.querySelectorAll('.city-item'))
      .find(el => el.textContent.trim() === 'Казань');

    item.click();

    expect(item.classList.contains('selected')).toBe(true);
    expect(document.getElementById('citySearch').value).toBe('Казань');
  });

  test('cityModal: клик вне окна закрывает модальное окно', () => {
    // Техника тест-дизайна: сценарий использования
    const cityModal = document.getElementById('cityModal');

    cityModal.dispatchEvent(new MouseEvent('click', {
      bubbles: true,
      clientX: 0,
      clientY: 0
    }));

    expect(cityModal.close).toHaveBeenCalled();
  });

  test('cityModal: Escape закрывает модальное окно', () => {
    // Техника тест-дизайна: переходы состояний
    const cityModal = document.getElementById('cityModal');

    cityModal.dispatchEvent(new KeyboardEvent('keydown', {
      bubbles: true,
      key: 'Escape'
    }));

    expect(cityModal.close).toHaveBeenCalled();
  });
});