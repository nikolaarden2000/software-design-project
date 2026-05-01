
'use strict';

const AUTH_PATH = './auth'; 

let auth;

function flushPromises(times = 5) {
  let chain = Promise.resolve();

  for (let i = 0; i < times; i += 1) {
    chain = chain.then(() => Promise.resolve());
  }

  return chain;
}

function setupDom() {
  document.body.innerHTML = `
    <div class="auth-card">
      <div class="auth-tabs">
        <button id="tab-login" class="active" type="button">Вход</button>
        <button id="tab-register" type="button">Регистрация</button>
      </div>

      <div id="authMessage"></div>

      <form id="loginForm">
        <input id="loginEmail" />
        <input id="loginPassword" />
        <button id="loginSubmit" type="submit">Войти</button>
      </form>

      <form id="registerForm" class="hidden">
        <input id="regUsername" />
        <input id="regEmail" />
        <input id="regPassword" />
        <input id="regConfirm" />
        <button id="registerSubmit" type="submit">Зарегистрироваться</button>
      </form>
    </div>
  `;
}

function setupApi() {
  window.Api = {
    getMe: jest.fn().mockResolvedValue({
      authenticated: false
    }),

    loginUser: jest.fn().mockResolvedValue({
      ok: true
    }),

    registerUser: jest.fn().mockResolvedValue({
      ok: true
    }),

    logoutUser: jest.fn().mockResolvedValue({
      ok: true
    })
  };
}

function createLogoutButton() {
  const logoutBtn = document.createElement('button');
  logoutBtn.id = 'logoutBtn';
  logoutBtn.type = 'button';
  logoutBtn.textContent = 'Выйти';
  document.body.appendChild(logoutBtn);

  return logoutBtn;
}

beforeEach(() => {
  jest.resetModules();
  jest.useFakeTimers();

  setupDom();
  setupApi();

  global.confirm = jest.fn();

  jest.spyOn(console, 'warn').mockImplementation(() => {});

  auth = require(AUTH_PATH);
});

afterEach(() => {
  jest.clearAllTimers();
  jest.useRealTimers();
  jest.restoreAllMocks();
});

describe('auth.js - модульные тесты', () => {
  /*
   * Техника тест-дизайна: smoke-тестирование.
   * Проверяем базовый сценарий инициализации модуля:
   * обработчики навешиваются, текущий пользователь проверяется.
   */
  test('initAuthModule: инициализирует модуль авторизации и проверяет пользователя', async () => {
    await auth.initAuthModule();

    expect(window.Api.getMe).toHaveBeenCalled();

    document.getElementById('tab-register').click();

    expect(document.getElementById('tab-register').classList.contains('active')).toBe(true);
    expect(document.getElementById('tab-login').classList.contains('active')).toBe(false);
    expect(document.getElementById('registerForm').classList.contains('hidden')).toBe(false);
    expect(document.getElementById('loginForm').classList.contains('hidden')).toBe(true);
  });

  /*
   * Техника тест-дизайна: негативное тестирование.
   * Проверяем защитное условие: если элемента authMessage нет, функция не падает.
   */
  test('setMessage: если authMessage отсутствует, функция завершается без ошибки', () => {
    document.getElementById('authMessage').remove();

    expect(() => auth.setMessage('Текст', 'error')).not.toThrow();
  });

  /*
   * Техника тест-дизайна: тестирование состояний и переходов.
   * Проверяем переход между вкладками "Вход" и "Регистрация".
   */
  test('bindTabs: переключает вкладки входа и регистрации', () => {
    const tabLogin = document.getElementById('tab-login');
    const tabRegister = document.getElementById('tab-register');
    const loginForm = document.getElementById('loginForm');
    const registerForm = document.getElementById('registerForm');

    auth.bindTabs(tabLogin, tabRegister, loginForm, registerForm);

    tabRegister.click();

    expect(tabRegister.classList.contains('active')).toBe(true);
    expect(tabLogin.classList.contains('active')).toBe(false);
    expect(registerForm.classList.contains('hidden')).toBe(false);
    expect(loginForm.classList.contains('hidden')).toBe(true);

    tabLogin.click();

    expect(tabLogin.classList.contains('active')).toBe(true);
    expect(tabRegister.classList.contains('active')).toBe(false);
    expect(loginForm.classList.contains('hidden')).toBe(false);
    expect(registerForm.classList.contains('hidden')).toBe(true);
  });

  /*
   * Техника тест-дизайна: негативное тестирование.
   * Проверяем защитное условие: если элементов вкладок нет, функция не падает.
   */
  test('bindTabs: если один из элементов отсутствует, функция не падает', () => {
    expect(() => auth.bindTabs(null, null, null, null)).not.toThrow();
  });

  /*
   * Техника тест-дизайна: классы эквивалентности.
   * Проверяем валидные и невалидные email-адреса как разные классы входных данных.
   */
  test('isValidEmail: определяет корректные и некорректные email', () => {
    const validEmails = [
      'user@example.com',
      'test.user@mail.ru',
      'student_123@university.edu'
    ];

    const invalidEmails = [
      '',
      'plain-text',
      'user@',
      '@example.com',
      'user@example',
      'user example@mail.ru'
    ];

    validEmails.forEach(email => {
      expect(auth.isValidEmail(email)).toBe(true);
    });

    invalidEmails.forEach(email => {
      expect(auth.isValidEmail(email)).toBe(false);
    });
  });

  /*
   * Техника тест-дизайна: классы эквивалентности.
   * Проверяем класс некорректных данных формы входа: пустой email или пароль.
   */
  test('bindLoginForm: при пустых полях показывает ошибку и не вызывает API', async () => {
    const loginForm = document.getElementById('loginForm');

    auth.bindLoginForm(loginForm);

    document.getElementById('loginEmail').value = '';
    document.getElementById('loginPassword').value = '';

    loginForm.dispatchEvent(
      new Event('submit', {
        bubbles: true,
        cancelable: true
      })
    );

    await flushPromises();

    expect(document.getElementById('authMessage').textContent).toBe('Введите email и пароль');
    expect(window.Api.loginUser).not.toHaveBeenCalled();
  });

  /*
   * Техника тест-дизайна: негативное тестирование.
   * Проверяем ошибочный ввод: email имеет неправильный формат.
   */
  test('bindLoginForm: при некорректном email показывает ошибку и не вызывает API', async () => {
    const loginForm = document.getElementById('loginForm');

    auth.bindLoginForm(loginForm);

    document.getElementById('loginEmail').value = 'wrong-email';
    document.getElementById('loginPassword').value = 'password123';

    loginForm.dispatchEvent(
      new Event('submit', {
        bubbles: true,
        cancelable: true
      })
    );

    await flushPromises();

    expect(document.getElementById('authMessage').textContent).toBe(
      'Пожалуйста, введите корректный адрес электронной почты'
    );
    expect(window.Api.loginUser).not.toHaveBeenCalled();
  });


  /*
   * Техника тест-дизайна: негативное тестирование.
   * Проверяем ошибку авторизации при неверном email или пароле.
   */
  test('bindLoginForm: при invalid_credentials показывает понятное сообщение', async () => {
    window.Api.loginUser.mockRejectedValueOnce({
      code: 'invalid_credentials'
    });

    const loginForm = document.getElementById('loginForm');

    auth.bindLoginForm(loginForm);

    document.getElementById('loginEmail').value = 'user@example.com';
    document.getElementById('loginPassword').value = 'wrong-password';

    loginForm.dispatchEvent(
      new Event('submit', {
        bubbles: true,
        cancelable: true
      })
    );

    await flushPromises();

    expect(document.getElementById('authMessage').textContent).toBe(
      'Неверный email или пароль'
    );
    expect(document.getElementById('loginSubmit').disabled).toBe(false);
  });

  /*
   * Техника тест-дизайна: негативное тестирование.
   * Проверяем защитное условие: если формы входа нет, функция не падает.
   */
  test('bindLoginForm: если loginForm отсутствует, функция не падает', () => {
    expect(() => auth.bindLoginForm(null)).not.toThrow();
  });

  /*
   * Техника тест-дизайна: классы эквивалентности.
   * Проверяем класс некорректных данных формы регистрации: не все поля заполнены.
   */
  test('bindRegisterForm: при пустых полях показывает ошибку и не вызывает API', async () => {
    const registerForm = document.getElementById('registerForm');

    auth.bindRegisterForm(registerForm);

    document.getElementById('regUsername').value = '';
    document.getElementById('regEmail').value = '';
    document.getElementById('regPassword').value = '';
    document.getElementById('regConfirm').value = '';

    registerForm.dispatchEvent(
      new Event('submit', {
        bubbles: true,
        cancelable: true
      })
    );

    await flushPromises();

    expect(document.getElementById('authMessage').textContent).toBe('Заполните все поля');
    expect(window.Api.registerUser).not.toHaveBeenCalled();
  });

  /*
   * Техника тест-дизайна: негативное тестирование.
   * Проверяем ошибочный формат email при регистрации.
   */
  test('bindRegisterForm: при некорректном email показывает ошибку и не вызывает API', async () => {
    const registerForm = document.getElementById('registerForm');

    auth.bindRegisterForm(registerForm);

    document.getElementById('regUsername').value = 'test-user';
    document.getElementById('regEmail').value = 'wrong-email';
    document.getElementById('regPassword').value = 'password123';
    document.getElementById('regConfirm').value = 'password123';

    registerForm.dispatchEvent(
      new Event('submit', {
        bubbles: true,
        cancelable: true
      })
    );

    await flushPromises();

    expect(document.getElementById('authMessage').textContent).toBe(
      'Пожалуйста, введите корректный адрес электронной почты'
    );
    expect(window.Api.registerUser).not.toHaveBeenCalled();
  });

  /*
   * Техника тест-дизайна: классы эквивалентности.
   * Проверяем границу совпадения паролей: два значения должны быть строго одинаковыми.
   */
  test('bindRegisterFormPassword: если пароль и подтверждение отличаются, показывает ошибку', async () => {
    const registerForm = document.getElementById('registerForm');

    auth.bindRegisterForm(registerForm);

    document.getElementById('regUsername').value = 'test-user';
    document.getElementById('regEmail').value = 'user@example.com';
    document.getElementById('regPassword').value = 'password123';
    document.getElementById('regConfirm').value = 'password124';

    registerForm.dispatchEvent(
      new Event('submit', {
        bubbles: true,
        cancelable: true
      })
    );

    await flushPromises();

    expect(document.getElementById('authMessage').textContent).toBe('Пароли не совпадают');
    expect(window.Api.registerUser).not.toHaveBeenCalled();
  });

  /*
   * Техника тест-дизайна: тестирование пользовательского сценария.
   * Проверяем успешную регистрацию пользователя и переход обратно на вкладку входа.
   */
  test('bindRegisterForm: при корректных данных вызывает registerUser и переключает на вкладку входа', async () => {
    const tabLogin = document.getElementById('tab-login');
    const tabRegister = document.getElementById('tab-register');
    const loginForm = document.getElementById('loginForm');
    const registerForm = document.getElementById('registerForm');
    const registerSubmit = document.getElementById('registerSubmit');

    auth.bindTabs(tabLogin, tabRegister, loginForm, registerForm);
    auth.bindRegisterForm(registerForm);

    tabRegister.click();

    document.getElementById('regUsername').value = 'test-user';
    document.getElementById('regEmail').value = ' user@example.com ';
    document.getElementById('regPassword').value = 'password123';
    document.getElementById('regConfirm').value = 'password123';

    registerForm.dispatchEvent(
      new Event('submit', {
        bubbles: true,
        cancelable: true
      })
    );

    expect(registerSubmit.disabled).toBe(true);
    expect(document.getElementById('authMessage').textContent).toBe('Отправка регистрации...');

    await flushPromises();

    expect(window.Api.registerUser).toHaveBeenCalledWith({
      username: 'test-user',
      email: 'user@example.com',
      password: 'password123'
    });

    expect(document.getElementById('authMessage').textContent).toBe(
      'Регистрация успешна. Пожалуйста, войдите в аккаунт.'
    );

    expect(registerSubmit.disabled).toBe(false);
    expect(tabLogin.classList.contains('active')).toBe(true);
    expect(loginForm.classList.contains('hidden')).toBe(false);
    expect(registerForm.classList.contains('hidden')).toBe(true);
  });

  /*
   * Техника тест-дизайна: негативное тестирование.
   * Проверяем ошибку регистрации, когда email уже занят.
   */
  test('bindRegisterForm: при email_already_exists показывает сообщение об ошибке', async () => {
    window.Api.registerUser.mockRejectedValueOnce({
      code: 'email_already_exists'
    });

    const registerForm = document.getElementById('registerForm');

    auth.bindRegisterForm(registerForm);

    document.getElementById('regUsername').value = 'test-user';
    document.getElementById('regEmail').value = 'user@example.com';
    document.getElementById('regPassword').value = 'password123';
    document.getElementById('regConfirm').value = 'password123';

    registerForm.dispatchEvent(
      new Event('submit', {
        bubbles: true,
        cancelable: true
      })
    );

    await flushPromises();

    expect(document.getElementById('authMessage').textContent).toBe(
      'Указанный email уже занят'
    );
    expect(document.getElementById('registerSubmit').disabled).toBe(false);
  });

  /*
   * Техника тест-дизайна: негативное тестирование.
   * Проверяем защитное условие: если формы регистрации нет, функция не падает.
   */
  test('bindRegisterForm: если registerForm отсутствует, функция не падает', () => {
    expect(() => auth.bindRegisterForm(null)).not.toThrow();
  });


  /*
   * Техника тест-дизайна: негативное тестирование.
   * Проверяем ошибку выхода из аккаунта.
   */
  test('bindLogoutButton: при ошибке logoutUser показывает сообщение ошибки', async () => {
    const logoutBtn = createLogoutButton();

    global.confirm.mockReturnValueOnce(true);

    window.Api.logoutUser.mockRejectedValueOnce({
      message: 'Ошибка сервера'
    });

    auth.bindLogoutButton(logoutBtn);

    logoutBtn.click();

    await flushPromises();

    expect(document.getElementById('authMessage').textContent).toBe('Ошибка сервера');
    expect(logoutBtn.disabled).toBe(false);
  });

  /*
   * Техника тест-дизайна: негативное тестирование.
   * Проверяем защитное условие: если кнопки выхода нет, функция не падает.
   */
  test('bindLogoutButton: если logoutBtn отсутствует, функция не падает', () => {
    expect(() => auth.bindLogoutButton(null)).not.toThrow();
  });

  /*
   * Техника тест-дизайна: тестирование состояний.
   * Проверяем состояние страницы, если пользователь уже авторизован.
   */
  test('checkAlreadyAuthenticated: если пользователь авторизован, показывает блок уже выполненного входа', async () => {
    window.Api.getMe.mockResolvedValueOnce({
      authenticated: true,
      user: {
        username: 'Slava'
      }
    });

    await auth.checkAlreadyAuthenticated();

    expect(document.querySelector('.auth-header').textContent).toBe('Вы уже авторизованы');
    expect(document.getElementById('authMessage').textContent).toBe(
      'Вы уже вошли в аккаунт как Slava.'
    );
    expect(document.querySelector('.link-as-btn.primary').getAttribute('href')).toBe('/me');
    expect(document.getElementById('logoutBtn').textContent).toBe('Выйти');
  });

  /*
   * Техника тест-дизайна: классы эквивалентности.
   * Проверяем вариант авторизованного пользователя без username.
   */
  test('renderAlreadyAuthenticated: если username отсутствует, показывает общее сообщение', () => {
    auth.renderAlreadyAuthenticated({});

    expect(document.querySelector('.auth-header').textContent).toBe('Вы уже авторизованы');
    expect(document.getElementById('authMessage').textContent).toBe(
      'Вы уже вошли в аккаунт.'
    );
  });

  /*
   * Техника тест-дизайна: негативное тестирование.
   * Проверяем ошибку проверки авторизации.
   */
  test('checkAlreadyAuthenticated: при ошибке getMe пишет предупреждение и не ломает страницу', async () => {
    window.Api.getMe.mockRejectedValueOnce(new Error('getMe failed'));

    await auth.checkAlreadyAuthenticated();

    expect(console.warn).toHaveBeenCalledWith(
      'Не удалось проверить авторизацию:',
      expect.any(Error)
    );

    expect(document.querySelector('.auth-card')).not.toBe(null);
  });

  /*
   * Техника тест-дизайна: негативное тестирование.
   * Проверяем защитное условие: если auth-card отсутствует, функция не падает.
   */
  test('renderAlreadyAuthenticated: если auth-card отсутствует, функция не падает', () => {
    document.querySelector('.auth-card').remove();

    expect(() => auth.renderAlreadyAuthenticated({ username: 'Slava' })).not.toThrow();
  });

  /*
   * Техника тест-дизайна: классы эквивалентности.
   * Проверяем разные классы ошибок: известный code, message и fallback-сообщение.
   */
  test('handleAuthError: выбирает сообщение по code, message или fallback', () => {
    const message = document.getElementById('authMessage');

    auth.handleAuthError(
      {
        code: 'invalid_credentials'
      },
      {
        invalid_credentials: 'Неверный email или пароль'
      }
    );

    expect(message.textContent).toBe('Неверный email или пароль');

    auth.handleAuthError(
      {
        message: 'Сервер временно недоступен'
      },
      {}
    );

    expect(message.textContent).toBe('Сервер временно недоступен');

    auth.handleAuthError({}, {});

    expect(message.textContent).toBe('Ошибка доступа к серверу. Повторите попытку.');
  });
});