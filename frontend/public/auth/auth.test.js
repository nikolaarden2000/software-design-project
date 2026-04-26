/**
 * auth.test.js
 * Тесты для auth.js
 */

const { setMessage, postJson, initAuthModule } = require('./auth.js');
async function flushPromises() {
  await Promise.resolve();
  await Promise.resolve();
  await Promise.resolve();
  await Promise.resolve();
}
describe('auth.js', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.spyOn(global, 'setTimeout');
    jest.spyOn(global, 'clearTimeout');

    document.body.innerHTML = '';
    window.__USER__ = {};
    global.fetch = jest.fn();
    global.confirm = jest.fn();
    delete window.location;
    window.location = { href: '', reload: jest.fn() };
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
    jest.restoreAllMocks();
  });

  function renderGuestAuthDOM() {
    document.body.innerHTML = `
      <div id="authMessage"></div>

      <button class="auth-tab active" id="tab-login" data-tab="login">Вход</button>
      <button class="auth-tab" id="tab-register" data-tab="register">Регистрация</button>

      <form id="loginForm" class="auth-form">
        <input id="loginEmail" type="email" />
        <input id="loginPassword" type="password" />
        <button type="submit" id="loginSubmit">Войти</button>
      </form>

      <form id="registerForm" class="auth-form hidden">
        <input id="regUsername" type="text" />
        <input id="regEmail" type="email" />
        <input id="regPassword" type="password" />
        <input id="regConfirm" type="password" />
        <button type="submit" id="registerSubmit">Зарегистрироваться</button>
      </form>
    `;
  }

  function renderAuthorizedDOM() {
    document.body.innerHTML = `
      <div id="authMessage"></div>
      <button id="logoutBtn">Выйти</button>
    `;
  }

  describe('setMessage', () => {
    test('setMessage: корректно устанавливает текст и цвет для ошибки', () => {
      // Техника тест-дизайна: классы эквивалентности
      document.body.innerHTML = `<div id="authMessage"></div>`;

      const el = document.getElementById('authMessage');
      setMessage('Ошибка', 'error', true);

      expect(el.textContent).toBe('Ошибка');
      expect(el.style.color).toBe('rgb(220, 38, 38)');
    });

    test('setMessage: для success задаёт успешный цвет', () => {
      // Техника тест-дизайна: классы эквивалентности
      document.body.innerHTML = `<div id="authMessage"></div>`;

      const el = document.getElementById('authMessage');
      setMessage('Успех', 'success', true);

      expect(el.textContent).toBe('Успех');
      expect(el.style.color).toBe('rgb(6, 95, 70)');
    });

    test('setMessage: автоматически очищает сообщение через 5000 мс', () => {
      // Техника тест-дизайна: граничные условия
      document.body.innerHTML = `<div id="authMessage"></div>`;

      const el = document.getElementById('authMessage');
      setMessage('Временное сообщение', 'info', false);

      expect(el.textContent).toBe('Временное сообщение');

      jest.advanceTimersByTime(4999);
      expect(el.textContent).toBe('Временное сообщение');

      jest.advanceTimersByTime(1);
      expect(el.textContent).toBe('');
    });

    test('setMessage: ничего не делает, если элемента authMessage нет', () => {
      // Техника тест-дизайна: предугадывание ошибок
      expect(() => setMessage('text', 'info', false)).not.toThrow();
    });
  });

  describe('postJson', () => {
    test('postJson: отправляет POST с JSON, credentials и body', async () => {
      // Техника тест-дизайна: классы эквивалентности
      const fakeResponse = { ok: true };
      global.fetch.mockResolvedValue(fakeResponse);

      const payload = { email: 'test@mail.com', password: '123456' };
      const result = await postJson('/api/login', payload);

      expect(global.fetch).toHaveBeenCalledWith('/api/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'same-origin',
        body: JSON.stringify(payload)
      });
      expect(result).toBe(fakeResponse);
    });
  });

  describe('initAuthModule - tabs', () => {
    test('initAuthModule: переключает вкладки login/register', () => {
      // Техника тест-дизайна: переходы состояний
      renderGuestAuthDOM();
      initAuthModule();

      const tabLogin = document.getElementById('tab-login');
      const tabRegister = document.getElementById('tab-register');
      const loginForm = document.getElementById('loginForm');
      const registerForm = document.getElementById('registerForm');

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

    test('initAuthModule: если пользователь уже авторизован, логика форм не инициализируется', () => {
      // Техника тест-дизайна: классы эквивалентности
      renderGuestAuthDOM();
      window.__USER__ = { auth: true };

      initAuthModule();

      const tabRegister = document.getElementById('tab-register');
      const registerForm = document.getElementById('registerForm');
      const loginForm = document.getElementById('loginForm');

      tabRegister.click();

      expect(registerForm.classList.contains('hidden')).toBe(true);
      expect(loginForm.classList.contains('hidden')).toBe(false);
    });
  });

  describe('initAuthModule - register', () => {
    test('register: невалидный email показывает ошибку и не отправляет запрос', async () => {
      // Техника тест-дизайна: классы эквивалентности
      renderGuestAuthDOM();
      initAuthModule();

      document.getElementById('regUsername').value = 'slava';
      document.getElementById('regEmail').value = 'wrong-email';
      document.getElementById('regPassword').value = '123456';
      document.getElementById('regConfirm').value = '123456';

      document.getElementById('registerForm').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));

      expect(document.getElementById('authMessage').textContent)
        .toBe('Пожалуйста, введите корректный адрес электронной почты (например, name@mail.com)');
      expect(global.fetch).not.toHaveBeenCalled();
    });

test('register: пустые поля показывают ошибку', () => {
  // Техника тест-дизайна: классы эквивалентности
  renderGuestAuthDOM();
  initAuthModule();

  document.getElementById('regUsername').value = '';
  document.getElementById('regEmail').value = 'test@mail.com';
  document.getElementById('regPassword').value = '123456';
  document.getElementById('regConfirm').value = '123456';

  document.getElementById('registerForm').dispatchEvent(
    new Event('submit', { bubbles: true, cancelable: true })
  );

  expect(document.getElementById('authMessage').textContent).toBe('Заполните все поля');
  expect(global.fetch).not.toHaveBeenCalled();
});

    test('register: несовпадение паролей показывает ошибку', async () => {
      // Техника тест-дизайна: попарное тестирование
      renderGuestAuthDOM();
      initAuthModule();

      document.getElementById('regUsername').value = 'slava';
      document.getElementById('regEmail').value = 'test@mail.com';
      document.getElementById('regPassword').value = '123456';
      document.getElementById('regConfirm').value = '654321';

      document.getElementById('registerForm').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));

      expect(document.getElementById('authMessage').textContent).toBe('Пароли не совпадают');
      expect(global.fetch).not.toHaveBeenCalled();
    });

 test('register: успешная регистрация переключает на вкладку входа', async () => {
  // Техника тест-дизайна: сценарий использования
  renderGuestAuthDOM();
  global.fetch.mockResolvedValue({ ok: true });
  initAuthModule();

  const tabLogin = document.getElementById('tab-login');

  document.getElementById('regUsername').value = 'slava';
  document.getElementById('regEmail').value = 'test@mail.com';
  document.getElementById('regPassword').value = '123456';
  document.getElementById('regConfirm').value = '123456';

  document.getElementById('tab-register').click();

  document.getElementById('registerForm').dispatchEvent(
    new Event('submit', { bubbles: true, cancelable: true })
  );

  await flushPromises();

  expect(global.fetch).toHaveBeenCalledWith('/api/register', expect.objectContaining({
    method: 'POST'
  }));
  expect(document.getElementById('authMessage').textContent)
    .toBe('Регистрация успешна. Пожалуйста, войдите в аккаунт.');
  expect(tabLogin.classList.contains('active')).toBe(true);
});

test('register: статус 409 показывает что email занят', async () => {
  // Техника тест-дизайна: анализ альтернативных ветвей
  renderGuestAuthDOM();
  global.fetch.mockResolvedValue({ ok: false, status: 409 });
  initAuthModule();

  document.getElementById('regUsername').value = 'slava';
  document.getElementById('regEmail').value = 'test@mail.com';
  document.getElementById('regPassword').value = '123456';
  document.getElementById('regConfirm').value = '123456';

  document.getElementById('registerForm').dispatchEvent(
    new Event('submit', { bubbles: true, cancelable: true })
  );

  await flushPromises();

  expect(document.getElementById('authMessage').textContent).toBe('Указанный email уже занят');
});

test('register: ошибка сети показывает сообщение об ошибке соединения', async () => {
  // Техника тест-дизайна: предугадывание ошибок
  renderGuestAuthDOM();
  global.fetch.mockRejectedValue(new Error('Network error'));
  initAuthModule();

  document.getElementById('regUsername').value = 'slava';
  document.getElementById('regEmail').value = 'test@mail.com';
  document.getElementById('regPassword').value = '123456';
  document.getElementById('regConfirm').value = '123456';

  document.getElementById('registerForm').dispatchEvent(
    new Event('submit', { bubbles: true, cancelable: true })
  );

  await flushPromises();

  expect(document.getElementById('authMessage').textContent).toBe('Ошибка соединения');
});

  describe('initAuthModule - login', () => {
    test('login: пустые email и пароль показывают ошибку', async () => {
      // Техника тест-дизайна: классы эквивалентности
      renderGuestAuthDOM();
      initAuthModule();

      document.getElementById('loginEmail').value = '';
      document.getElementById('loginPassword').value = '';

      document.getElementById('loginForm').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));

      expect(document.getElementById('authMessage').textContent).toBe('Введите email и пароль');
      expect(global.fetch).not.toHaveBeenCalled();
    });

    test('login: невалидный email показывает ошибку', async () => {
      // Техника тест-дизайна: классы эквивалентности
      renderGuestAuthDOM();
      initAuthModule();

      document.getElementById('loginEmail').value = 'bad-email';
      document.getElementById('loginPassword').value = '123456';

      document.getElementById('loginForm').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));

      expect(document.getElementById('authMessage').textContent)
        .toBe('Пожалуйста, введите корректный адрес электронной почты (например, name@mail.com)');
      expect(global.fetch).not.toHaveBeenCalled();
    });

test('login: успешный вход делает редирект на главную через 500 мс', async () => {
  // Техника тест-дизайна: граничные условия
  renderGuestAuthDOM();
  global.fetch.mockResolvedValue({ ok: true });
  initAuthModule();

  document.getElementById('loginEmail').value = 'test@mail.com';
  document.getElementById('loginPassword').value = '123456';

  document.getElementById('loginForm').dispatchEvent(
    new Event('submit', { bubbles: true, cancelable: true })
  );

  await flushPromises();

  expect(document.getElementById('authMessage').textContent).toBe('Вход успешен. Перенаправление...');
  expect(window.location.href).toBe('');

  jest.advanceTimersByTime(499);
  expect(window.location.href).toBe('');

  jest.advanceTimersByTime(1);
  expect(window.location.href).toBe('/');
});

test('login: статус 401 показывает сообщение о неверных данных', async () => {
  // Техника тест-дизайна: таблица решений
  renderGuestAuthDOM();
  global.fetch.mockResolvedValue({ ok: false, status: 401 });
  initAuthModule();

  document.getElementById('loginEmail').value = 'test@mail.com';
  document.getElementById('loginPassword').value = 'wrong';

  document.getElementById('loginForm').dispatchEvent(
    new Event('submit', { bubbles: true, cancelable: true })
  );

  await flushPromises();

  expect(document.getElementById('authMessage').textContent).toBe('Неверный email или пароль');
});

test('login: ошибка сети показывает сообщение об ошибке соединения', async () => {
  // Техника тест-дизайна: предугадывание ошибок
  renderGuestAuthDOM();
  global.fetch.mockRejectedValue(new Error('Network error'));
  initAuthModule();

  document.getElementById('loginEmail').value = 'test@mail.com';
  document.getElementById('loginPassword').value = '123456';

  document.getElementById('loginForm').dispatchEvent(
    new Event('submit', { bubbles: true, cancelable: true })
  );

  await flushPromises();

  expect(document.getElementById('authMessage').textContent).toBe('Ошибка соединения');
});
  });

  describe('initAuthModule - logout', () => {
    test('logout: отмена confirm не отправляет запрос', async () => {
      // Техника тест-дизайна: классы эквивалентности
      renderAuthorizedDOM();
      window.__USER__ = { auth: false };
      global.confirm.mockReturnValue(false);

      initAuthModule();
      document.getElementById('logoutBtn').click();

      expect(global.fetch).not.toHaveBeenCalled();
    });

    test('logout: успешный выход делает редирект через 600 мс', async () => {
      // Техника тест-дизайна: граничные условия
      renderAuthorizedDOM();
      window.__USER__ = { auth: false };
      global.confirm.mockReturnValue(true);
      global.fetch.mockResolvedValue({ ok: true });

      initAuthModule();
      document.getElementById('logoutBtn').click();

      await Promise.resolve();
      await Promise.resolve();

      expect(document.getElementById('authMessage').textContent)
        .toBe('Вы вышли из аккаунта. Перенаправление на главную...');
      expect(window.location.href).toBe('');

      jest.advanceTimersByTime(599);
      expect(window.location.href).toBe('');

      jest.advanceTimersByTime(1);
      expect(window.location.href).toBe('/');
    });

    test('logout: статус 400 показывает что сессия отсутствует', async () => {
      // Техника тест-дизайна: таблица решений
      renderAuthorizedDOM();
      window.__USER__ = { auth: false };
      global.confirm.mockReturnValue(true);
      global.fetch.mockResolvedValue({ ok: false, status: 400 });

      initAuthModule();
      document.getElementById('logoutBtn').click();

      await Promise.resolve();
      await Promise.resolve();

      expect(document.getElementById('authMessage').textContent).toBe('Сессия отсутствует или истекла');
    });

    test('logout: ошибка сети показывает сообщение об ошибке соединения', async () => {
      // Техника тест-дизайна: предугадывание ошибок
      renderAuthorizedDOM();
      window.__USER__ = { auth: false };
      global.confirm.mockReturnValue(true);
      global.fetch.mockRejectedValue(new Error('Network error'));

      initAuthModule();
      document.getElementById('logoutBtn').click();

      await Promise.resolve();
      await Promise.resolve();

      expect(document.getElementById('authMessage').textContent).toBe('Ошибка соединения');
    });
  });
});
});