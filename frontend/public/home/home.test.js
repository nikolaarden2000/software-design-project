

'use strict';

const HOME_PATH = './home'; 

let home;

function createRoom(overrides = {}) {
  return {
    id: 1,
    title: 'Переговорная',
    company: 'Company A',
    address: 'ул. Тестовая, 1',
    capacity: 4,
    price: 1000,
    image: '/room.png',
    ...overrides
  };
}

function flushPromises(times = 5) {
  let chain = Promise.resolve();

  for (let i = 0; i < times; i += 1) {
    chain = chain.then(() => Promise.resolve());
  }

  return chain;
}

function setupDom(initialCity = 'Москва') {
  document.body.dataset.initialCity = initialCity;

  document.body.innerHTML = `
    <a id="brand" href="/">Своя Бронь</a>

    <button id="authButton" type="button"></button>

    <button id="cityBtn" type="button">Выбрать город</button>
    <span id="cityName"></span>

    <dialog id="cityModal">
      <input id="citySearch" />
      <ul id="cityList">
        <li class="city-item">Москва</li>
        <li class="city-item">Санкт-Петербург</li>
        <li class="city-item">Казань</li>
        <li class="city-item">Екатеринбург</li>
        <li class="city-item">Новосибирск</li>
        <li class="city-item">Нижний Новгород</li>
      </ul>
      <button id="cityOk" type="button">OK</button>
      <button id="cityCancel" type="button">Cancel</button>
    </dialog>

    <div id="cardsWrapper"></div>
    <div id="statusBar" hidden></div>

    <div id="companyList"></div>
    <div id="companyToggleWrap"></div>

    <input id="priceInput" />
    <input id="capacityInput" />

    <button id="applyFilters" type="button"></button>
    <button id="clearFilters" type="button"></button>
  `;

  const cityModal = document.getElementById('cityModal');

  cityModal.showModal = jest.fn(() => {
    cityModal.open = true;
  });

  cityModal.close = jest.fn(() => {
    cityModal.open = false;
  });

  cityModal.getBoundingClientRect = jest.fn(() => ({
    left: 0,
    right: 100,
    top: 0,
    bottom: 100
  }));
}

function setupApi() {
  window.Api = {
    getMe: jest.fn().mockResolvedValue({
      authenticated: false
    }),

    getRoomFilters: jest.fn().mockResolvedValue({
      companies: ['Company A', 'Company B']
    }),

    getRooms: jest.fn().mockResolvedValue({
      items: [
        createRoom({
          id: 1,
          company: 'Company A'
        })
      ],
      pagination: {
        next_after_id: 1,
        has_more: false
      }
    })
  };
}

function setupIntersectionObserverMock() {
  class MockIntersectionObserver {
    static instances = [];

    constructor(callback, options) {
      this.callback = callback;
      this.options = options;
      this.observe = jest.fn();
      this.disconnect = jest.fn();

      MockIntersectionObserver.instances.push(this);
    }

    trigger(entries) {
      this.callback(entries);
    }
  }

  global.IntersectionObserver = MockIntersectionObserver;
}

beforeEach(() => {
  jest.resetModules();
  jest.useFakeTimers();

  setupDom();
  setupApi();
  setupIntersectionObserverMock();

  jest.spyOn(console, 'warn').mockImplementation(() => {});
  jest.spyOn(console, 'error').mockImplementation(() => {});

  home = require(HOME_PATH);
});

afterEach(() => {
  jest.runOnlyPendingTimers();
  jest.useRealTimers();
  jest.restoreAllMocks();
});

describe('home.js - модульные тесты', () => {
  /*
   * Техника тест-дизайна: smoke-тестирование.
   * Проверяем основной успешный сценарий инициализации страницы:
   * загрузка пользователя, фильтров и карточек.
   */
  test('init: инициализирует страницу, загружает фильтры и карточки', async () => {
    await home.init();

    expect(document.getElementById('cityName').textContent).toBe('Москва');

    expect(window.Api.getMe).toHaveBeenCalled();
    expect(window.Api.getRoomFilters).toHaveBeenCalledWith('Москва');
    expect(window.Api.getRooms).toHaveBeenCalledWith({
      city: 'Москва',
      limit: 100,
      after_id: 0
    });

    expect(document.getElementById('authButton').textContent).toBe('Войти');
    expect(document.getElementById('applyFilters').disabled).toBe(false);
    expect(document.getElementById('clearFilters').disabled).toBe(false);

    expect(document.querySelectorAll('.card')).toHaveLength(1);
    expect(document.getElementById('statusBar').hidden).toBe(true);
  });

  /*
   * Техника тест-дизайна: классы эквивалентности.
   * Проверяем разные классы пользователей:
   * superuser, admin и обычный пользователь.
   */
  test('loadCurrentUser: корректно отображает кнопку для разных ролей пользователя', async () => {
    const cases = [
      {
        role: 'superuser',
        expectedText: 'Панель платформы'
      },
      {
        role: 'admin',
        expectedText: 'Админ-панель'
      },
      {
        role: 'user',
        expectedText: 'Кабинет'
      }
    ];

    for (const item of cases) {
      window.Api.getMe.mockResolvedValueOnce({
        authenticated: true,
        user: {
          role: item.role,
          username: 'test-user'
        }
      });

      await home.loadCurrentUser();

      const authBtn = document.getElementById('authButton');

      expect(authBtn.dataset.auth).toBe('1');
      expect(authBtn.dataset.role).toBe(item.role);
      expect(authBtn.dataset.username).toBe('test-user');
      expect(authBtn.textContent).toBe(item.expectedText);
    }
  });

  /*
   * Техника тест-дизайна: негативное тестирование.
   * Проверяем ситуацию, когда API пользователя недоступен.
   */
  test('loadCurrentUser: при ошибке API считает пользователя неавторизованным', async () => {
    window.Api.getMe.mockRejectedValueOnce(new Error('API error'));

    await home.loadCurrentUser();

    const authBtn = document.getElementById('authButton');

    expect(home.__getIsAuthenticatedForTests()).toBe(false);
    expect(home.__getCurrentUserRoleForTests()).toBe(null);
    expect(authBtn.dataset.auth).toBe('0');
    expect(authBtn.dataset.role).toBe('');
    expect(authBtn.textContent).toBe('Войти');
  });

  /*
   * Техника тест-дизайна: классы эквивалентности.
   * Проверяем некорректный формат ответа фильтров.
   */
  test('loadFilters: если companies не массив, список компаний становится пустым', async () => {
    window.Api.getRoomFilters.mockResolvedValueOnce({
      companies: 'Company A'
    });

    await home.loadFilters();

    expect(home.__getAllCompaniesForTests()).toEqual([]);
    expect(document.getElementById('companyList').textContent).toContain('Нет данных');
    expect(document.getElementById('applyFilters').disabled).toBe(false);
  });

  /*
   * Техника тест-дизайна: негативное тестирование.
   * Проверяем отказ API фильтров.
   */
  test('loadFilters: при ошибке загрузки фильтров компании очищаются', async () => {
    window.Api.getRoomFilters.mockRejectedValueOnce(new Error('filters failed'));

    await home.loadFilters();

    expect(home.__getAllCompaniesForTests()).toEqual([]);
    expect(document.getElementById('companyList').textContent).toContain('Нет данных');
    expect(document.getElementById('applyFilters').disabled).toBe(false);
  });

  /*
   * Техника тест-дизайна: анализ граничных значений.
   * Проверяем ровно 100 элементов, то есть значение на границе BATCH_SIZE.
   */
  test('loadMoreRooms: если получено 100 карточек, hasMore остаётся true', async () => {
    const rooms = Array.from({ length: 100 }, (_, index) =>
      createRoom({
        id: index + 1,
        title: `Room ${index + 1}`,
        company: index % 2 === 0 ? 'Company A' : 'Company B'
      })
    );

    window.Api.getRooms.mockResolvedValueOnce(rooms);

    await home.loadMoreRooms(true);

    expect(home.__getAllItemsForTests()).toHaveLength(100);
    expect(home.__getLastAfterIdForTests()).toBe(100);
    expect(home.__getHasMoreForTests()).toBe(true);
    expect(document.querySelectorAll('.card')).toHaveLength(100);
    expect(document.getElementById('sentinel')).not.toBe(null);
  });

  /*
   * Техника тест-дизайна: классы эквивалентности.
   * Проверяем пустой список помещений как отдельный класс ответа.
   */
  test('loadMoreRooms: если помещений нет, показывает сообщение о пустом городе', async () => {
    window.Api.getRooms.mockResolvedValueOnce({
      items: [],
      pagination: {
        has_more: false
      }
    });

    await home.loadMoreRooms(true);

    expect(document.getElementById('cardsWrapper').textContent).toContain(
      'В этом городе нет помещений для бронирования'
    );
    expect(document.getElementById('sentinel')).toBe(null);
  });

  /*
   * Техника тест-дизайна: негативное тестирование.
   * Проверяем ошибку загрузки помещений при пустом каталоге.
   */
  test('loadMoreRooms: при ошибке загрузки и пустом каталоге показывает кнопку повтора', async () => {
    window.Api.getRooms.mockRejectedValueOnce(new Error('rooms failed'));

    await home.loadMoreRooms(true);

    const cardsWrapper = document.getElementById('cardsWrapper');

    expect(cardsWrapper.textContent).toContain(
      'Не удалось загрузить помещения. Попробуйте позже.'
    );
    expect(cardsWrapper.textContent).toContain('Попробовать сейчас');
  });

  /*
   * Техника тест-дизайна: тестирование состояний.
   * Проверяем ошибку загрузки, когда часть данных уже есть.
   */
  test('loadMoreRooms: при ошибке и уже загруженных карточках скрывает статус загрузки', async () => {
    home.__setAllItemsForTests([createRoom()]);
    window.Api.getRooms.mockRejectedValueOnce(new Error('next page failed'));

    await home.loadMoreRooms(false);

    expect(document.getElementById('statusBar').hidden).toBe(true);
  });

  /*
   * Техника тест-дизайна: тестирование состояний и переходов.
   * Проверяем защитные условия: загрузка уже идёт или данных больше нет.
   */
  test('loadMoreRooms: не делает запрос, если уже идёт загрузка или hasMore=false', async () => {
    home.__setIsLoadingForTests(true);

    await home.loadMoreRooms(false);

    expect(window.Api.getRooms).not.toHaveBeenCalled();

    home.__setIsLoadingForTests(false);
    home.__setHasMoreForTests(false);

    await home.loadMoreRooms(false);

    expect(window.Api.getRooms).not.toHaveBeenCalled();
  });

  /*
   * Техника тест-дизайна: тестирование состояний.
   * Проверяем сброс состояния каталога.
   */
  test('resetCatalogState: сбрасывает данные, фильтрацию и отключает observer', () => {
    const observer = {
      disconnect: jest.fn()
    };

    home.__setAllItemsForTests([createRoom()]);
    home.__setAllCompaniesForTests(['Company A']);
    home.__setLastAfterIdForTests(10);
    home.__setHasMoreForTests(false);
    home.__setIsLoadingForTests(true);
    home.__setFilteringForTests(true);
    home.__setInfiniteScrollObserverForTests(observer);

    home.resetCatalogState();

    expect(home.__getAllItemsForTests()).toEqual([]);
    expect(home.__getAllCompaniesForTests()).toEqual([]);
    expect(home.__getLastAfterIdForTests()).toBe(0);
    expect(home.__getHasMoreForTests()).toBe(true);
    expect(home.__getIsLoadingForTests()).toBe(false);
    expect(home.__getFilteringForTests()).toBe(false);
    expect(observer.disconnect).toHaveBeenCalled();
    expect(document.getElementById('companyList').textContent).toContain('Загрузка');
  });

  /*
   * Техника тест-дизайна: негативное тестирование.
   * Проверяем ввод города, которого нет в допустимом списке.
   */
  test('setCity: для несуществующего города показывает ошибку', () => {
    home.setCity('Лондон');

    expect(home.__getCityForTests()).toBe('Москва');
    expect(document.getElementById('cityError').textContent).toBe(
      'Такого города не существует'
    );
  });

  /*
   * Техника тест-дизайна: тестирование пользовательского сценария.
   * Проверяем смену города и перезагрузку каталога.
   */
  test('setCity: при корректном новом городе обновляет город и очищает фильтры', async () => {
    document.getElementById('priceInput').value = '1000';
    document.getElementById('capacityInput').value = '3';

    home.setCity('Казань');

    expect(home.__getCityForTests()).toBe('Казань');
    expect(document.getElementById('cityName').textContent).toBe('Казань');
    expect(document.body.dataset.initialCity).toBe('Казань');
    expect(document.getElementById('priceInput').value).toBe('');
    expect(document.getElementById('capacityInput').value).toBe('');

    await flushPromises();

    expect(window.Api.getRoomFilters).toHaveBeenCalledWith('Казань');
    expect(window.Api.getRooms).toHaveBeenCalled();
  });

  /*
   * Техника тест-дизайна: тестирование UI-состояний.
   * Проверяем открытие и закрытие модального окна выбора города.
   */
  test('openCityModal и closeCityModal: открывают и закрывают модальное окно', () => {
    const cityModal = document.getElementById('cityModal');

    home.openCityModal();

    expect(cityModal.showModal).toHaveBeenCalled();
    expect(document.getElementById('citySearch').value).toBe('Москва');
    expect(document.querySelector('.city-item.selected').textContent).toBe('Москва');

    home.closeCityModal();

    expect(cityModal.close).toHaveBeenCalled();
  });

  /*
   * Техника тест-дизайна: классы эквивалентности.
   * Проверяем поиск города: есть совпадение и нет совпадений.
   */
  test('filterCityList: фильтрует города и показывает "Город не найден"', () => {
    home.filterCityList('каз');

    const kazan = Array.from(document.querySelectorAll('.city-item'))
      .find(item => item.textContent === 'Казань');

    const moscow = Array.from(document.querySelectorAll('.city-item'))
      .find(item => item.textContent === 'Москва');

    expect(kazan.style.display).toBe('block');
    expect(moscow.style.display).toBe('none');

    home.filterCityList('ГородКоторогоНет');

    expect(document.querySelector('[data-no-results="1"]').textContent).toBe(
      'Город не найден'
    );
  });

  /*
   * Техника тест-дизайна: тестирование пользовательского сценария.
   * Проверяем выбор города из списка в модальном окне.
   */
  test('bindEvents: клик по городу выбирает его и записывает в поле поиска', () => {
    home.bindEvents();

    const kazan = Array.from(document.querySelectorAll('.city-item'))
      .find(item => item.textContent === 'Казань');

    kazan.dispatchEvent(new MouseEvent('click', { bubbles: true }));

    expect(kazan.classList.contains('selected')).toBe(true);
    expect(document.getElementById('citySearch').value).toBe('Казань');
  });

  /*
   * Техника тест-дизайна: негативное тестирование.
   * Проверяем Enter в пустом поле поиска города.
   */
  test('bindEvents: Enter в пустом поле поиска города показывает ошибку', () => {
    home.bindEvents();

    const citySearch = document.getElementById('citySearch');

    citySearch.value = '';

    citySearch.dispatchEvent(
      new KeyboardEvent('keyup', {
        key: 'Enter',
        bubbles: true
      })
    );

    expect(document.getElementById('cityError').textContent).toBe(
      'Введите название города'
    );
  });

  /*
   * Техника тест-дизайна: негативное тестирование.
   * Проверяем кнопку OK без выбранного города и без текста поиска.
   */
  test('bindEvents: OK без выбранного города показывает ошибку', () => {
    home.bindEvents();

    document
      .querySelectorAll('.city-item')
      .forEach(item => item.classList.remove('selected'));

    document.getElementById('citySearch').value = '';

    document.getElementById('cityOk').click();

    expect(document.getElementById('cityError').textContent).toBe(
      'Выберите город из списка или введите название'
    );
  });

  /*
   * Техника тест-дизайна: тестирование UI-состояний.
   * Проверяем закрытие модального окна по Escape и клику вне области.
   */
  test('bindEvents: закрывает модальное окно по Escape и клику вне области', () => {
    home.bindEvents();

    const cityModal = document.getElementById('cityModal');

    cityModal.dispatchEvent(
      new KeyboardEvent('keydown', {
        key: 'Escape',
        bubbles: true
      })
    );

    expect(cityModal.close).toHaveBeenCalledTimes(1);

    cityModal.dispatchEvent(
      new MouseEvent('click', {
        clientX: 150,
        clientY: 150,
        bubbles: true
      })
    );

    expect(cityModal.close).toHaveBeenCalledTimes(2);
  });

  /*
   * Техника тест-дизайна: тестирование состояний.
   * Проверяем включение и отключение фильтров.
   */
  test('setFiltersEnabled: включает и отключает кнопки и чекбоксы фильтров', () => {
    home.__setAllCompaniesForTests(['Company A']);
    home.__setFiltersEnabledForTests(true);
    home.refreshFilterValues();

    home.setFiltersEnabled(false);

    expect(document.getElementById('applyFilters').disabled).toBe(true);
    expect(document.getElementById('clearFilters').disabled).toBe(true);
    expect(document.querySelector('input[type="checkbox"]').disabled).toBe(true);

    home.setFiltersEnabled(true);

    expect(document.getElementById('applyFilters').disabled).toBe(false);
    expect(document.getElementById('clearFilters').disabled).toBe(false);
    expect(document.querySelector('input[type="checkbox"]').disabled).toBe(false);
  });

  /*
   * Техника тест-дизайна: анализ граничных значений.
   * Проверяем границу отображения кнопки "Показать всё": больше 6 компаний.
   */
  test('refreshFilterValues: при количестве компаний больше 6 показывает кнопку раскрытия', () => {
    home.__setFiltersEnabledForTests(true);
    home.__setAllCompaniesForTests([
      'Company 1',
      'Company 2',
      'Company 3',
      'Company 4',
      'Company 5',
      'Company 6',
      'Company 7'
    ]);

    home.refreshFilterValues();

    const toggleBtn = document.querySelector('#companyToggleWrap button');

    expect(toggleBtn.textContent).toBe('Показать всё');
    expect(document.getElementById('companyList').style.maxHeight).toBe('160px');

    toggleBtn.click();

    expect(toggleBtn.textContent).toBe('Свернуть');
    expect(document.getElementById('companyList').style.maxHeight).toBe('360px');

    toggleBtn.click();

    expect(toggleBtn.textContent).toBe('Показать всё');
    expect(document.getElementById('companyList').style.maxHeight).toBe('160px');
  });

  /*
   * Техника тест-дизайна: попарное тестирование.
   * Проверяем совместную работу нескольких фильтров:
   * цена + вместимость + компания.
   */
  test('applyFilters: фильтрует карточки по цене, вместимости и компании', () => {
    home.__setAllItemsForTests([
      createRoom({
        id: 1,
        title: 'Room 1',
        company: 'Company A',
        price: 500,
        capacity: 2
      }),
      createRoom({
        id: 2,
        title: 'Room 2',
        company: 'Company B',
        price: 900,
        capacity: 4
      }),
      createRoom({
        id: 3,
        title: 'Room 3',
        company: 'Company B',
        price: 1200,
        capacity: 5
      })
    ]);

    home.__setAllCompaniesForTests(['Company A', 'Company B']);
    home.__setFiltersEnabledForTests(true);
    home.refreshFilterValues();

    document.getElementById('priceInput').value = '1000';
    document.getElementById('capacityInput').value = '3';

    const companyBCheckbox = Array.from(
      document.querySelectorAll('#companyList input[type="checkbox"]')
    ).find(cb => cb.value === 'Company B');

    companyBCheckbox.checked = true;

    const result = home.applyFilters(true);

    expect(result).toHaveLength(1);
    expect(result[0].id).toBe(2);
    expect(home.__getFilteringForTests()).toBe(true);
    expect(document.querySelectorAll('.card')).toHaveLength(1);
    expect(document.getElementById('cardsWrapper').textContent).toContain('Room 2');
  });

  /*
   * Техника тест-дизайна: классы эквивалентности.
   * Проверяем фильтрацию при пустом массиве карточек.
   */
  test('applyFilters: при пустом каталоге возвращает пустой массив и отображает пустой результат', () => {
    home.__setAllItemsForTests([]);

    const result = home.applyFilters(true);

    expect(result).toEqual([]);
    expect(document.getElementById('cardsWrapper').textContent).toContain(
      'Ничего не найдено по выбранным фильтрам'
    );
  });

  /*
   * Техника тест-дизайна: тестирование пользовательского сценария.
   * Проверяем очистку фильтров кнопкой "Сбросить".
   */
  test('bindEvents: кнопка очистки фильтров сбрасывает поля и чекбоксы', () => {
    home.__setAllItemsForTests([
      createRoom({
        id: 1,
        company: 'Company A'
      })
    ]);

    home.__setAllCompaniesForTests(['Company A']);
    home.__setFiltersEnabledForTests(true);
    home.refreshFilterValues();
    home.bindEvents();

    document.getElementById('priceInput').value = '1000';
    document.getElementById('capacityInput').value = '3';
    document.querySelector('#companyList input[type="checkbox"]').checked = true;

    document.getElementById('clearFilters').click();

    expect(document.getElementById('priceInput').value).toBe('');
    expect(document.getElementById('capacityInput').value).toBe('');
    expect(document.querySelector('#companyList input[type="checkbox"]').checked).toBe(false);
    expect(home.__getFilteringForTests()).toBe(false);
  });

  /*
   * Техника тест-дизайна: классы эквивалентности.
   * Проверяем ввод цены: цифры, буквы, ведущие нули.
   */
  test('bindEvents: поле цены удаляет нецифровые символы и лишний ведущий ноль', () => {
    home.bindEvents();

    const priceInput = document.getElementById('priceInput');

    priceInput.value = '00a12b';

    priceInput.dispatchEvent(
      new Event('input', {
        bubbles: true
      })
    );

    expect(priceInput.value).toBe('012');
  });

  /*
   * Техника тест-дизайна: анализ граничных значений.
   * Проверяем blur у цены: отрицательное значение и положительное значение.
   */
  test('bindEvents: blur у цены нормализует значение', () => {
    home.bindEvents();

    const priceInput = document.getElementById('priceInput');

    priceInput.value = '-5';

    priceInput.dispatchEvent(
      new Event('blur', {
        bubbles: true
      })
    );

    expect(priceInput.value).toBe('');

    priceInput.value = '500';

    priceInput.dispatchEvent(
      new Event('blur', {
        bubbles: true
      })
    );

    expect(priceInput.value).toBe('500');
  });

  /*
   * Техника тест-дизайна: негативное тестирование.
   * Проверяем запрет ввода недопустимых клавиш в поле цены.
   */
  test('bindEvents: поле цены запрещает ввод букв с клавиатуры', () => {
    home.bindEvents();

    const priceInput = document.getElementById('priceInput');

    const invalidEvent = new KeyboardEvent('keydown', {
      key: 'a',
      bubbles: true,
      cancelable: true
    });

    priceInput.dispatchEvent(invalidEvent);

    expect(invalidEvent.defaultPrevented).toBe(true);

    const validEvent = new KeyboardEvent('keydown', {
      key: '5',
      bubbles: true,
      cancelable: true
    });

    priceInput.dispatchEvent(validEvent);

    expect(validEvent.defaultPrevented).toBe(false);
  });

  /*
   * Техника тест-дизайна: классы эквивалентности.
   * Проверяем поле вместимости: цифры остаются, буквы удаляются.
   */
  test('bindEvents: поле вместимости оставляет только цифры', () => {
    home.bindEvents();

    const capacityInput = document.getElementById('capacityInput');

    capacityInput.value = '12abc3';

    capacityInput.dispatchEvent(
      new Event('input', {
        bubbles: true
      })
    );

    expect(capacityInput.value).toBe('123');
  });

  /*
   * Техника тест-дизайна: классы эквивалентности.
   * Проверяем renderCards для пустого массива.
   */
  test('renderCards: для пустого массива показывает сообщение о пустом результате', () => {
    home.renderCards([], true);

    expect(document.getElementById('cardsWrapper').textContent).toContain(
      'Ничего не найдено по выбранным фильтрам'
    );
    expect(document.getElementById('sentinel')).toBe(null);
  });

  /*
   * Техника тест-дизайна: негативное тестирование.
   * Проверяем карточку без id.
   */
  test('renderCards: при клике по карточке без id пишет предупреждение', () => {
    home.renderCards([
      createRoom({
        id: null,
        title: 'Комната без id'
      })
    ]);

    document.querySelector('.card').click();

    expect(console.warn).toHaveBeenCalledWith(
      'id отсутствует у карточки',
      expect.objectContaining({
        title: 'Комната без id'
      })
    );
  });

  /*
   * Техника тест-дизайна: тестирование состояний и переходов.
   * Проверяем появление sentinel и автозагрузку следующей порции данных.
   */
  test('ensureSentinel: при пересечении sentinel запускает загрузку следующей страницы', async () => {
    window.Api.getRooms.mockResolvedValueOnce({
      items: [],
      pagination: {
        has_more: false
      }
    });

    home.__setHasMoreForTests(true);
    home.__setIsLoadingForTests(false);
    home.__setFilteringForTests(false);

    home.ensureSentinel(true);

    const observer = global.IntersectionObserver.instances.at(-1);

    observer.trigger([
      {
        isIntersecting: true
      }
    ]);

    await flushPromises();

    expect(window.Api.getRooms).toHaveBeenCalledWith({
      city: 'Москва',
      limit: 100,
      after_id: 0
    });
  });

  /*
   * Техника тест-дизайна: тестирование состояний.
   * Проверяем отключение sentinel и observer.
   */
  test('ensureSentinel: при enable=false удаляет sentinel и отключает observer', () => {
    home.ensureSentinel(true);

    const observer = global.IntersectionObserver.instances.at(-1);

    expect(document.getElementById('sentinel')).not.toBe(null);

    home.ensureSentinel(false);

    expect(observer.disconnect).toHaveBeenCalled();
    expect(document.getElementById('sentinel')).toBe(null);
    expect(home.__getInfiniteScrollObserverForTests()).toBe(null);
  });

  /*
   * Техника тест-дизайна: негативное тестирование.
   * Проверяем сообщение об ошибке с кнопкой повторной загрузки.
   */
  test('showCenteredMessage: кнопка повтора запускает loadInitialData', async () => {
    home.showCenteredMessage('Ошибка загрузки', true);

    const retryButton = document.querySelector('.center-message button');

    expect(retryButton.textContent).toBe('Попробовать сейчас');

    retryButton.click();

    await flushPromises();

    expect(window.Api.getRoomFilters).toHaveBeenCalled();
    expect(window.Api.getRooms).toHaveBeenCalled();
  });

  /*
   * Техника тест-дизайна: тестирование UI-состояний.
   * Проверяем отображение и скрытие statusBar.
   */
  test('showStatusBar и hideStatusBar: показывают и скрывают статус загрузки', () => {
    home.showStatusBar('Загрузка...');

    expect(document.getElementById('statusBar').hidden).toBe(false);
    expect(document.getElementById('statusBar').textContent).toBe('Загрузка...');

    home.hideStatusBar();

    expect(document.getElementById('statusBar').hidden).toBe(true);
    expect(document.getElementById('statusBar').textContent).toBe('');
  });

  /*
   * Техника тест-дизайна: тестирование ошибок и исключительных ситуаций.
   * Проверяем глобальный обработчик window.error.
   */
  test('bindEvents: глобальная ошибка показывает пользователю сообщение', () => {
    home.bindEvents();

    window.dispatchEvent(
      new ErrorEvent('error', {
        message: 'Unexpected error',
        error: new Error('Unexpected error')
      })
    );

    expect(document.getElementById('cardsWrapper').textContent).toContain(
      'Произошла ошибка. Обновите страницу или попробуйте позже.'
    );
  });
});