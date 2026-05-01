
const superuserModule = require('./superuser.js');

describe('superuser.js', () => {
  async function flushPromises() {
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
  }

  function buildBaseDOM() {
    document.body.innerHTML = `
      <button id="logoutBtn">Выйти</button>

      <section id="accessDenied" class="hidden">Нет доступа</section>
      <section id="superuserContent" class="hidden">
        <div id="companiesList"></div>
        <div id="locationsList"></div>
        <div id="adminsList"></div>
        <div id="moderationList"></div>

        <form id="companyForm">
          <input id="companyName" />
          <textarea id="companyDescription"></textarea>
          <button type="submit">Создать компанию</button>
        </form>

        <form id="locationForm">
          <select id="locationCompany"></select>
          <input id="locationCity" />
          <input id="locationAddress" />
          <input id="locationLat" />
          <input id="locationLng" />
          <input id="locationTimezone" value="Europe/Moscow" />
          <button type="submit">Создать локацию</button>
        </form>

        <form id="adminForm">
          <input id="adminUsername" />
          <input id="adminEmail" />
          <input id="adminPassword" />
          <button type="submit">Создать администратора</button>
        </form>

        <form id="assignLocationsForm">
          <select id="assignAdmin"></select>
          <select id="assignLocations" multiple></select>
          <div id="assignmentLocationRows"></div>
          <button type="submit">Привязать</button>
        </form>
      </section>

      <div id="locationPickerModal" class="hidden" aria-hidden="true">
        <button data-close-location-modal type="button">Закрыть</button>
        <input id="locationPickerSearch" />
        <div id="locationPickerList"></div>
      </div>
    `;
  }

  function makeApi(overrides = {}) {
    return {
      getMe: jest.fn().mockResolvedValue({
        authenticated: true,
        user: { role: 'superuser' }
      }),
      getCompanies: jest.fn().mockResolvedValue({ items: [] }),
      getLocations: jest.fn().mockResolvedValue({ items: [] }),
      getAdmins: jest.fn().mockResolvedValue({ items: [] }),
      getModerationRooms: jest.fn().mockResolvedValue({ items: [] }),

      createCompany: jest.fn().mockResolvedValue({}),
      createLocation: jest.fn().mockResolvedValue({}),
      createAdmin: jest.fn().mockResolvedValue({}),
      assignAdminToLocation: jest.fn().mockResolvedValue({}),

      approveRoom: jest.fn().mockResolvedValue({}),
      rejectRoom: jest.fn().mockResolvedValue({}),
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
      href: ''
    };

    window.Api = makeApi();

    buildBaseDOM();

    superuserModule.__resetStateForTests();
  });

  test('escapeHtml: экранирует HTML-символы', () => {
    // Техника тест-дизайна: классы эквивалентности
    expect(superuserModule.escapeHtml('<script>"x"&\'y\'</script>'))
      .toBe('&lt;script&gt;&quot;x&quot;&amp;&#039;y&#039;&lt;/script&gt;');
  });

  test('showAccessDenied: показывает блок отказа и скрывает контент', () => {
    // Техника тест-дизайна: переходы состояний
    superuserModule.showAccessDenied();

    expect(document.getElementById('accessDenied').classList.contains('hidden')).toBe(false);
    expect(document.getElementById('superuserContent').classList.contains('hidden')).toBe(true);
  });

  test('showContent: скрывает отказ в доступе и показывает контент', () => {
    // Техника тест-дизайна: переходы состояний
    superuserModule.showContent();

    expect(document.getElementById('accessDenied').classList.contains('hidden')).toBe(true);
    expect(document.getElementById('superuserContent').classList.contains('hidden')).toBe(false);
  });

  test('initSuperuserPage: неавторизованный пользователь перенаправляется на /auth', async () => {
    // Техника тест-дизайна: классы эквивалентности
    window.Api = makeApi({
      getMe: jest.fn().mockResolvedValue({ authenticated: false })
    });

    await superuserModule.initSuperuserPage();

    expect(window.location.href).toBe('/auth');
  });

  test('initSuperuserPage: пользователь без роли superuser видит отказ в доступе', async () => {
    // Техника тест-дизайна: классы эквивалентности
    window.Api = makeApi({
      getMe: jest.fn().mockResolvedValue({
        authenticated: true,
        user: { role: 'admin' }
      })
    });

    await superuserModule.initSuperuserPage();

    expect(document.getElementById('accessDenied').classList.contains('hidden')).toBe(false);
    expect(document.getElementById('superuserContent').classList.contains('hidden')).toBe(true);
  });

  test('initSuperuserPage: ошибка getMe показывает alert', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    window.Api = makeApi({
      getMe: jest.fn().mockRejectedValue(new Error('Ошибка загрузки панели'))
    });

    await superuserModule.initSuperuserPage();

    expect(global.alert).toHaveBeenCalledWith('Ошибка загрузки панели');
    expect(console.error).toHaveBeenCalled();
  });

  test('initSuperuserPage: для superuser загружает все данные и показывает контент', async () => {
    // Техника тест-дизайна: сценарий использования
    window.Api = makeApi({
      getCompanies: jest.fn().mockResolvedValue({ items: [] }),
      getLocations: jest.fn().mockResolvedValue({ items: [] }),
      getAdmins: jest.fn().mockResolvedValue({ items: [] }),
      getModerationRooms: jest.fn().mockResolvedValue({ items: [] })
    });

    await superuserModule.initSuperuserPage();

    expect(document.getElementById('accessDenied').classList.contains('hidden')).toBe(true);
    expect(document.getElementById('superuserContent').classList.contains('hidden')).toBe(false);
    expect(window.Api.getCompanies).toHaveBeenCalled();
    expect(window.Api.getLocations).toHaveBeenCalled();
    expect(window.Api.getAdmins).toHaveBeenCalled();
    expect(window.Api.getModerationRooms).toHaveBeenCalled();
  });

  test('renderLimitedList: при пустом списке показывает текст пустого состояния', () => {
    // Техника тест-дизайна: классы эквивалентности
    const root = document.getElementById('companiesList');

    superuserModule.renderLimitedList(
      root,
      [],
      'companies',
      'Компаний пока нет',
      item => `<div>${item.name}</div>`
    );

    expect(root.textContent).toContain('Компаний пока нет');
  });

  test('renderCompanies: при пустом списке показывает сообщение', () => {
    // Техника тест-дизайна: классы эквивалентности
    superuserModule.__setCompaniesForTests([]);
    superuserModule.renderCompanies();

    expect(document.getElementById('companiesList').textContent).toContain('Компаний пока нет');
  });

  test('renderCompanies: при количестве больше лимита показывает кнопку "Показать все"', () => {
    // Техника тест-дизайна: граничные условия
    superuserModule.__setCompaniesForTests([
      { id: 1, name: 'A', description: 'd1', locations_count: 1 },
      { id: 2, name: 'B', description: 'd2', locations_count: 2 },
      { id: 3, name: 'C', description: 'd3', locations_count: 3 },
      { id: 4, name: 'D', description: 'd4', locations_count: 4 }
    ]);

    superuserModule.renderCompanies();

    expect(document.getElementById('companiesList').textContent).toContain('Показать все (4)');
  });

  test('renderCompanies: кнопка toggle раскрывает и сворачивает список', () => {
    // Техника тест-дизайна: переходы состояний
    superuserModule.__setCompaniesForTests([
      { id: 1, name: 'A', description: 'd1', locations_count: 1 },
      { id: 2, name: 'B', description: 'd2', locations_count: 2 },
      { id: 3, name: 'C', description: 'd3', locations_count: 3 },
      { id: 4, name: 'D', description: 'd4', locations_count: 4 }
    ]);

    superuserModule.renderCompanies();

    const button = document.querySelector('[data-list-toggle="companies"]');
    expect(button).not.toBeNull();

    button.click();
    expect(document.getElementById('companiesList').textContent).toContain('Свернуть');

    const collapseButton = document.querySelector('[data-list-toggle="companies"]');
    collapseButton.click();
    expect(document.getElementById('companiesList').textContent).toContain('Показать все (4)');
  });

  test('renderLocations: при пустом списке показывает сообщение', () => {
    // Техника тест-дизайна: классы эквивалентности
    superuserModule.__setLocationsForTests([]);
    superuserModule.renderLocations();

    expect(document.getElementById('locationsList').textContent).toContain('Локаций пока нет');
  });

  test('renderLocations: отображает список локаций', () => {
    // Техника тест-дизайна: сценарий использования
    superuserModule.__setLocationsForTests([
      { id: 1, company_name: 'Компания А', city: 'Москва', address: 'Адрес 1', timezone: 'Europe/Moscow' }
    ]);

    superuserModule.renderLocations();

    expect(document.getElementById('locationsList').textContent).toContain('Компания А');
    expect(document.getElementById('locationsList').textContent).toContain('Москва');
  });

  test('renderAdmins: при пустом списке показывает сообщение', () => {
    // Техника тест-дизайна: классы эквивалентности
    superuserModule.__setAdminsForTests([]);
    superuserModule.renderAdmins();

    expect(document.getElementById('adminsList').textContent).toContain('Администраторов пока нет');
  });

  test('renderAdmins: отображает администратора и его локации', () => {
    // Техника тест-дизайна: сценарий использования
    superuserModule.__setAdminsForTests([
      {
        id: 1,
        username: 'admin1',
        email: 'admin1@test.com',
        locations: [
          { company_name: 'Компания А', address: 'Адрес 1' },
          { company_name: 'Компания Б', address: 'Адрес 2' }
        ]
      }
    ]);

    superuserModule.renderAdmins();

    expect(document.getElementById('adminsList').textContent).toContain('admin1');
    expect(document.getElementById('adminsList').textContent).toContain('admin1@test.com');
    expect(document.getElementById('adminsList').textContent).toContain('Компания А');
    expect(document.getElementById('adminsList').textContent).toContain('Компания Б');
  });

  test('renderModerationRooms: при пустом списке показывает сообщение', () => {
    // Техника тест-дизайна: классы эквивалентности
    superuserModule.__setModerationRoomsForTests([]);
    superuserModule.renderModerationRooms();

    expect(document.getElementById('moderationList').textContent).toContain('Нет помещений на модерации');
  });

  test('renderModerationRooms: отображает карточку помещения и кнопки модерации', () => {
    // Техника тест-дизайна: сценарий использования
    superuserModule.__setModerationRoomsForTests([
      {
        id: 101,
        title: 'Комната 101',
        company_name: 'Компания А',
        city: 'Москва',
        address: 'Адрес 1',
        description: 'Описание',
        price: 1000,
        capacity: 10,
        available_from: '09:00',
        available_to: '21:00',
        status: 'pending',
        created_by: {
          username: 'user1',
          email: 'user1@test.com'
        }
      }
    ]);

    superuserModule.renderModerationRooms();

    expect(document.getElementById('moderationList').textContent).toContain('Комната 101');
    expect(document.querySelector('[data-action="approve"]')).not.toBeNull();
    expect(document.querySelector('[data-action="reject"]')).not.toBeNull();
  });

  test('fillCompanySelect: заполняет select компаниями', () => {
    // Техника тест-дизайна: сценарий использования
    superuserModule.__setCompaniesForTests([
      { id: 1, name: 'Компания А' },
      { id: 2, name: 'Компания Б' }
    ]);

    superuserModule.fillCompanySelect();

    const options = document.querySelectorAll('#locationCompany option');
    expect(options.length).toBe(3);
    expect(options[1].textContent).toBe('Компания А');
    expect(options[2].textContent).toBe('Компания Б');
  });

  test('fillAdminSelect: сохраняет текущее выбранное значение', () => {
    // Техника тест-дизайна: переходы состояний
    superuserModule.__setAdminsForTests([
      { id: 1, username: 'admin1', email: 'a1@test.com' },
      { id: 2, username: 'admin2', email: 'a2@test.com' }
    ]);

    const select = document.getElementById('assignAdmin');
    select.innerHTML = `<option value="2" selected>old</option>`;
    select.value = '2';

    superuserModule.fillAdminSelect();

    expect(select.value).toBe('2');
  });

  test('fillLocationsMultiSelect: заполняет multiple-select локациями', () => {
    // Техника тест-дизайна: сценарий использования
    superuserModule.__setLocationsForTests([
      { id: 10, company_name: 'Компания А', city: 'Москва', address: 'Адрес 1' },
      { id: 11, company_name: 'Компания Б', city: 'Казань', address: 'Адрес 2' }
    ]);

    superuserModule.fillLocationsMultiSelect();

    const options = document.querySelectorAll('#assignLocations option');
    expect(options.length).toBe(2);
    expect(options[0].value).toBe('10');
    expect(options[1].value).toBe('11');
  });

  test('bindCompanyForm: пустое название не позволяет создать компанию', async () => {
    // Техника тест-дизайна: классы эквивалентности
    superuserModule.bindCompanyForm();

    document.getElementById('companyName').value = '';
    document.getElementById('companyDescription').value = 'Описание';

    document.getElementById('companyForm').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();

    expect(global.alert).toHaveBeenCalledWith('Введите название компании');
    expect(window.Api.createCompany).not.toHaveBeenCalled();
  });

  test('bindCompanyForm: успешное создание компании сбрасывает форму и обновляет список', async () => {
    // Техника тест-дизайна: сценарий использования
    window.Api = makeApi({
      createCompany: jest.fn().mockResolvedValue({}),
      getCompanies: jest.fn().mockResolvedValue({
        items: [{ id: 1, name: 'Компания А', description: 'Описание', locations_count: 0 }]
      })
    });

    superuserModule.bindCompanyForm();

    document.getElementById('companyName').value = 'Компания А';
    document.getElementById('companyDescription').value = 'Описание';

    document.getElementById('companyForm').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();

    expect(window.Api.createCompany).toHaveBeenCalledWith({
      name: 'Компания А',
      description: 'Описание'
    });
    expect(document.getElementById('companiesList').textContent).toContain('Компания А');
  });

  test('bindCompanyForm: ошибка createCompany показывает alert', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    window.Api = makeApi({
      createCompany: jest.fn().mockRejectedValue(new Error('Ошибка создания компании'))
    });

    superuserModule.bindCompanyForm();

    document.getElementById('companyName').value = 'Компания А';
    document.getElementById('companyDescription').value = 'Описание';

    document.getElementById('companyForm').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();

    expect(global.alert).toHaveBeenCalledWith('Ошибка создания компании');
    expect(console.error).toHaveBeenCalled();
  });

  test('bindLocationForm: без company/city/address не позволяет создать локацию', async () => {
    // Техника тест-дизайна: классы эквивалентности
    superuserModule.bindLocationForm();

    document.getElementById('locationCompany').innerHTML = '<option value="">Выберите компанию</option>';
    document.getElementById('locationCompany').value = '';
    document.getElementById('locationCity').value = '';
    document.getElementById('locationAddress').value = '';

    document.getElementById('locationForm').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();

    expect(global.alert).toHaveBeenCalledWith('Заполните компанию, город и адрес');
    expect(window.Api.createLocation).not.toHaveBeenCalled();
  });

  test('bindLocationForm: успешное создание локации сбрасывает форму и обновляет списки', async () => {
    // Техника тест-дизайна: сценарий использования
    window.Api = makeApi({
      createLocation: jest.fn().mockResolvedValue({}),
      getLocations: jest.fn().mockResolvedValue({
        items: [
          { id: 1, company_name: 'Компания А', city: 'Москва', address: 'Адрес 1', timezone: 'Europe/Moscow' }
        ]
      })
    });

    superuserModule.bindLocationForm();

    document.getElementById('locationCompany').innerHTML = `
      <option value="">Выберите компанию</option>
      <option value="5">Компания А</option>
    `;
    document.getElementById('locationCompany').value = '5';
    document.getElementById('locationCity').value = 'Москва';
    document.getElementById('locationAddress').value = 'Адрес 1';
    document.getElementById('locationLat').value = '55.75';
    document.getElementById('locationLng').value = '37.61';
    document.getElementById('locationTimezone').value = 'Europe/Moscow';

    document.getElementById('locationForm').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();

    expect(window.Api.createLocation).toHaveBeenCalledWith({
      company_id: 5,
      city: 'Москва',
      address: 'Адрес 1',
      lat: 55.75,
      lng: 37.61,
      timezone: 'Europe/Moscow'
    });
    expect(document.getElementById('locationsList').textContent).toContain('Москва');
    expect(document.getElementById('locationTimezone').value).toBe('Europe/Moscow');
  });

  test('bindLocationForm: ошибка createLocation показывает alert', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    window.Api = makeApi({
      createLocation: jest.fn().mockRejectedValue(new Error('Ошибка создания локации'))
    });

    superuserModule.bindLocationForm();

    document.getElementById('locationCompany').innerHTML = `<option value="5">Компания А</option>`;
    document.getElementById('locationCompany').value = '5';
    document.getElementById('locationCity').value = 'Москва';
    document.getElementById('locationAddress').value = 'Адрес 1';

    document.getElementById('locationForm').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();

    expect(global.alert).toHaveBeenCalledWith('Ошибка создания локации');
    expect(console.error).toHaveBeenCalled();
  });

  test('bindAdminForm: без username/email/password не позволяет создать администратора', async () => {
    // Техника тест-дизайна: классы эквивалентности
    superuserModule.bindAdminForm();

    document.getElementById('adminUsername').value = '';
    document.getElementById('adminEmail').value = '';
    document.getElementById('adminPassword').value = '';

    document.getElementById('adminForm').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();

    expect(global.alert).toHaveBeenCalledWith('Заполните имя, email и пароль администратора');
    expect(window.Api.createAdmin).not.toHaveBeenCalled();
  });

  test('bindAdminForm: успешное создание администратора обновляет список и показывает alert', async () => {
    // Техника тест-дизайна: сценарий использования
    window.Api = makeApi({
      createAdmin: jest.fn().mockResolvedValue({}),
      getAdmins: jest.fn().mockResolvedValue({
        items: [
          { id: 1, username: 'admin1', email: 'admin1@test.com', locations: [] }
        ]
      })
    });

    superuserModule.bindAdminForm();

    document.getElementById('adminUsername').value = 'admin1';
    document.getElementById('adminEmail').value = 'admin1@test.com';
    document.getElementById('adminPassword').value = '123456';

    document.getElementById('adminForm').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();

    expect(window.Api.createAdmin).toHaveBeenCalledWith({
      username: 'admin1',
      email: 'admin1@test.com',
      password: '123456'
    });
    expect(document.getElementById('adminsList').textContent).toContain('admin1');
    expect(global.alert).toHaveBeenCalledWith('Администратор создан. Теперь можно привязать к нему локации.');
  });

  test('bindAdminForm: ошибка createAdmin показывает alert', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    window.Api = makeApi({
      createAdmin: jest.fn().mockRejectedValue(new Error('Ошибка создания администратора'))
    });

    superuserModule.bindAdminForm();

    document.getElementById('adminUsername').value = 'admin1';
    document.getElementById('adminEmail').value = 'admin1@test.com';
    document.getElementById('adminPassword').value = '123456';

    document.getElementById('adminForm').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();

    expect(global.alert).toHaveBeenCalledWith('Ошибка создания администратора');
    expect(console.error).toHaveBeenCalled();
  });

  test('bindAssignLocationsForm: без администратора показывает alert', async () => {
    // Техника тест-дизайна: классы эквивалентности
    superuserModule.__setSelectedAssignmentLocationIdsForTests([10]);
    superuserModule.bindAssignLocationsForm();

    document.getElementById('assignAdmin').innerHTML = '<option value="">Выберите администратора</option>';
    document.getElementById('assignAdmin').value = '';

    document.getElementById('assignLocationsForm').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();

    expect(global.alert).toHaveBeenCalledWith('Выберите администратора');
    expect(window.Api.assignAdminToLocation).not.toHaveBeenCalled();
  });

  test('bindAssignLocationsForm: без выбранных локаций показывает alert', async () => {
    // Техника тест-дизайна: классы эквивалентности
    superuserModule.__setSelectedAssignmentLocationIdsForTests([]);
    superuserModule.bindAssignLocationsForm();

    document.getElementById('assignAdmin').innerHTML = '<option value="1">admin1</option>';
    document.getElementById('assignAdmin').value = '1';

    document.getElementById('assignLocationsForm').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();

    expect(global.alert).toHaveBeenCalledWith('Выберите хотя бы одну локацию');
    expect(window.Api.assignAdminToLocation).not.toHaveBeenCalled();
  });

  test('bindAssignLocationsForm: успешно привязывает уникальные локации к администратору', async () => {
    // Техника тест-дизайна: классы эквивалентности
    window.Api = makeApi({
      assignAdminToLocation: jest.fn().mockResolvedValue({}),
      getAdmins: jest.fn().mockResolvedValue({
        items: [{ id: 1, username: 'admin1', email: 'admin1@test.com', locations: [] }]
      })
    });

    superuserModule.__setSelectedAssignmentLocationIdsForTests([10, 11, 10]);
    superuserModule.bindAssignLocationsForm();

    document.getElementById('assignAdmin').innerHTML = '<option value="1">admin1</option>';
    document.getElementById('assignAdmin').value = '1';

    document.getElementById('assignLocationsForm').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();

    expect(window.Api.assignAdminToLocation).toHaveBeenCalledTimes(2);
    expect(window.Api.assignAdminToLocation).toHaveBeenCalledWith(1, 10);
    expect(window.Api.assignAdminToLocation).toHaveBeenCalledWith(1, 11);
    expect(superuserModule.__getSelectedAssignmentLocationIdsForTests()).toEqual([]);
    expect(global.alert).toHaveBeenCalledWith('Локации успешно привязаны к администратору');
  });

  test('bindAssignLocationsForm: ошибка привязки показывает alert', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    window.Api = makeApi({
      assignAdminToLocation: jest.fn().mockRejectedValue(new Error('Ошибка привязки локаций'))
    });

    superuserModule.__setSelectedAssignmentLocationIdsForTests([10]);
    superuserModule.bindAssignLocationsForm();

    document.getElementById('assignAdmin').innerHTML = '<option value="1">admin1</option>';
    document.getElementById('assignAdmin').value = '1';

    document.getElementById('assignLocationsForm').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();

    expect(global.alert).toHaveBeenCalledWith('Ошибка привязки локаций');
    expect(console.error).toHaveBeenCalled();
  });

  test('approveRoom: успешно удаляет помещение из moderationRooms и перерисовывает список', async () => {
    // Техника тест-дизайна: сценарий использования
    superuserModule.__setModerationRoomsForTests([
      {
        id: 101,
        title: 'Комната 101',
        company_name: 'Компания А',
        city: 'Москва',
        address: 'Адрес 1',
        description: 'Описание',
        price: 1000,
        capacity: 10,
        available_from: '09:00',
        available_to: '21:00',
        status: 'pending',
        created_by: { username: 'user1', email: 'user1@test.com' }
      }
    ]);
    superuserModule.renderModerationRooms();

    await superuserModule.approveRoom('101');

    expect(window.Api.approveRoom).toHaveBeenCalledWith('101');
    expect(superuserModule.__getModerationRoomsForTests()).toEqual([]);
    expect(document.getElementById('moderationList').textContent).toContain('Нет помещений на модерации');
  });

  test('approveRoom: ошибка approveRoom показывает alert', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    window.Api = makeApi({
      approveRoom: jest.fn().mockRejectedValue(new Error('Ошибка одобрения помещения'))
    });

    await superuserModule.approveRoom('101');

    expect(global.alert).toHaveBeenCalledWith('Ошибка одобрения помещения');
    expect(console.error).toHaveBeenCalled();
  });

  test('rejectRoom: без причины отклонения показывает alert', async () => {
    // Техника тест-дизайна: классы эквивалентности
    document.getElementById('moderationList').innerHTML = `
      <textarea id="rejectReason-101"></textarea>
    `;

    await superuserModule.rejectRoom('101');

    expect(global.alert).toHaveBeenCalledWith('Введите причину отклонения');
    expect(window.Api.rejectRoom).not.toHaveBeenCalled();
  });

  test('rejectRoom: успешно отклоняет помещение и удаляет его из moderationRooms', async () => {
    // Техника тест-дизайна: сценарий использования
    superuserModule.__setModerationRoomsForTests([
      {
        id: 101,
        title: 'Комната 101',
        company_name: 'Компания А',
        city: 'Москва',
        address: 'Адрес 1',
        description: 'Описание',
        price: 1000,
        capacity: 10,
        available_from: '09:00',
        available_to: '21:00',
        status: 'pending',
        created_by: { username: 'user1', email: 'user1@test.com' }
      }
    ]);
    superuserModule.renderModerationRooms();

    document.getElementById('rejectReason-101').value = 'Недостаточно данных';

    await superuserModule.rejectRoom('101');

    expect(window.Api.rejectRoom).toHaveBeenCalledWith('101', 'Недостаточно данных');
    expect(superuserModule.__getModerationRoomsForTests()).toEqual([]);
  });

  test('rejectRoom: ошибка rejectRoom показывает alert', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    window.Api = makeApi({
      rejectRoom: jest.fn().mockRejectedValue(new Error('Ошибка отклонения помещения'))
    });

    document.getElementById('moderationList').innerHTML = `
      <textarea id="rejectReason-101">Причина</textarea>
    `;

    await superuserModule.rejectRoom('101');

    expect(global.alert).toHaveBeenCalledWith('Ошибка отклонения помещения');
    expect(console.error).toHaveBeenCalled();
  });

  test('renderAssignmentLocationRows: рендерит выбранные локации и пустую строку выбора', () => {
    // Техника тест-дизайна: сценарий использования
    superuserModule.__setLocationsForTests([
      { id: 10, company_name: 'Компания А', city: 'Москва', address: 'Адрес 1' },
      { id: 11, company_name: 'Компания Б', city: 'Казань', address: 'Адрес 2' }
    ]);
    superuserModule.__setSelectedAssignmentLocationIdsForTests([10]);

    superuserModule.renderAssignmentLocationRows();

    expect(document.getElementById('assignmentLocationRows').textContent).toContain('Компания А');
    expect(document.getElementById('assignmentLocationRows').textContent).toContain('Выбрать локацию');
  });

  test('removeAssignmentLocation: удаляет выбранную локацию по индексу', () => {
    // Техника тест-дизайна: сценарий использования
    superuserModule.__setLocationsForTests([
      { id: 10, company_name: 'Компания А', city: 'Москва', address: 'Адрес 1' },
      { id: 11, company_name: 'Компания Б', city: 'Казань', address: 'Адрес 2' }
    ]);
    superuserModule.__setSelectedAssignmentLocationIdsForTests([10, 11]);

    superuserModule.removeAssignmentLocation(0);

    expect(superuserModule.__getSelectedAssignmentLocationIdsForTests()).toEqual([11]);
  });

  test('openLocationPicker: открывает модалку и устанавливает активный индекс строки', () => {
    // Техника тест-дизайна: переходы состояний
    superuserModule.__setLocationsForTests([
      { id: 10, company_name: 'Компания А', city: 'Москва', address: 'Адрес 1', timezone: 'Europe/Moscow' }
    ]);

    superuserModule.openLocationPicker(0);

    expect(superuserModule.__getActiveLocationPickerRowIndexForTests()).toBe(0);
    expect(document.getElementById('locationPickerModal').classList.contains('hidden')).toBe(false);
    expect(document.getElementById('locationPickerModal').getAttribute('aria-hidden')).toBe('false');
  });

  test('closeLocationPicker: закрывает модалку и сбрасывает активный индекс', () => {
    // Техника тест-дизайна: переходы состояний
    superuserModule.__setActiveLocationPickerRowIndexForTests(2);
    document.getElementById('locationPickerModal').classList.remove('hidden');
    document.getElementById('locationPickerModal').setAttribute('aria-hidden', 'false');

    superuserModule.closeLocationPicker();

    expect(superuserModule.__getActiveLocationPickerRowIndexForTests()).toBeNull();
    expect(document.getElementById('locationPickerModal').classList.contains('hidden')).toBe(true);
    expect(document.getElementById('locationPickerModal').getAttribute('aria-hidden')).toBe('true');
  });

  test('renderLocationPickerList: при отсутствии совпадений показывает сообщение', () => {
    // Техника тест-дизайна: классы эквивалентности
    superuserModule.__setLocationsForTests([
      { id: 10, company_name: 'Компания А', city: 'Москва', address: 'Адрес 1', timezone: 'Europe/Moscow' }
    ]);

    superuserModule.renderLocationPickerList('Тверь');

    expect(document.getElementById('locationPickerList').textContent).toContain('Локации не найдены');
  });

  test('renderLocationPickerList: помечает уже выбранную в другой строке локацию как disabled', () => {
    // Техника тест-дизайна: классы эквивалентности
    superuserModule.__setLocationsForTests([
      { id: 10, company_name: 'Компания А', city: 'Москва', address: 'Адрес 1', timezone: 'Europe/Moscow' },
      { id: 11, company_name: 'Компания Б', city: 'Казань', address: 'Адрес 2', timezone: 'Europe/Moscow' }
    ]);
    superuserModule.__setSelectedAssignmentLocationIdsForTests([10]);
    superuserModule.__setActiveLocationPickerRowIndexForTests(1);

    superuserModule.renderLocationPickerList('');

    const disabledOption = document.querySelector('[data-location-id="10"]');
    expect(disabledOption.disabled).toBe(true);
    expect(disabledOption.textContent).toContain('Уже выбрана');
  });

  test('selectLocationForAssignment: при activeRowIndex=null ничего не делает', () => {
    // Техника тест-дизайна: классы эквивалентности
    superuserModule.__setSelectedAssignmentLocationIdsForTests([]);
    superuserModule.__setActiveLocationPickerRowIndexForTests(null);

    superuserModule.selectLocationForAssignment(10);

    expect(superuserModule.__getSelectedAssignmentLocationIdsForTests()).toEqual([]);
  });

  test('selectLocationForAssignment: добавляет новую локацию в конец списка', () => {
    // Техника тест-дизайна: сценарий использования
    superuserModule.__setLocationsForTests([
      { id: 10, company_name: 'Компания А', city: 'Москва', address: 'Адрес 1' },
      { id: 11, company_name: 'Компания Б', city: 'Казань', address: 'Адрес 2' }
    ]);
    superuserModule.__setSelectedAssignmentLocationIdsForTests([10]);
    superuserModule.__setActiveLocationPickerRowIndexForTests(1);

    superuserModule.selectLocationForAssignment(11);

    expect(superuserModule.__getSelectedAssignmentLocationIdsForTests()).toEqual([10, 11]);
    expect(superuserModule.__getActiveLocationPickerRowIndexForTests()).toBeNull();
  });

  test('selectLocationForAssignment: заменяет локацию в существующей строке', () => {
    // Техника тест-дизайна: сценарий использования
    superuserModule.__setLocationsForTests([
      { id: 10, company_name: 'Компания А', city: 'Москва', address: 'Адрес 1' },
      { id: 11, company_name: 'Компания Б', city: 'Казань', address: 'Адрес 2' }
    ]);
    superuserModule.__setSelectedAssignmentLocationIdsForTests([10]);
    superuserModule.__setActiveLocationPickerRowIndexForTests(0);

    superuserModule.selectLocationForAssignment(11);

    expect(superuserModule.__getSelectedAssignmentLocationIdsForTests()).toEqual([11]);
  });


  test('bindLogout: успешный logout ведёт на главную', async () => {
    // Техника тест-дизайна: сценарий использования
    superuserModule.bindLogout();

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

    superuserModule.bindLogout();

    document.getElementById('logoutBtn').click();
    await flushPromises();

    expect(global.alert).toHaveBeenCalledWith('Ошибка выхода');
    expect(console.error).toHaveBeenCalled();
  });

  test('bindLogout: если кнопки logout нет, функция завершается без ошибки', () => {
    // Техника тест-дизайна: предугадывание ошибок
    document.getElementById('logoutBtn').remove();

    expect(() => superuserModule.bindLogout()).not.toThrow();
  });

});