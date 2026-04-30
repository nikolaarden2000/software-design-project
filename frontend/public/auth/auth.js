
  'use strict';

  document.addEventListener('DOMContentLoaded', initAuthModule);

  function setMessage(text, type = 'info', sticky = false) {
    const el = document.getElementById('authMessage');

    if (!el) return;

    el.textContent = text || '';

    if (type === 'error') {
      el.style.color = '#dc2626';
    } else if (type === 'success') {
      el.style.color = '#065f46';
    } else {
      el.style.color = 'var(--muted)';
    }

    if (!sticky) {
      clearTimeout(el._hideTimer);

      el._hideTimer = setTimeout(() => {
        el.textContent = '';
        el.style.color = 'var(--muted)';
      }, 5000);
    }
  }

  async function initAuthModule() {
    const tabLogin = document.getElementById('tab-login');
    const tabRegister = document.getElementById('tab-register');
    const loginForm = document.getElementById('loginForm');
    const registerForm = document.getElementById('registerForm');
    const logoutBtn = document.getElementById('logoutBtn');

    bindTabs(tabLogin, tabRegister, loginForm, registerForm);
    bindLoginForm(loginForm);
    bindRegisterForm(registerForm);
    bindLogoutButton(logoutBtn);

    await checkAlreadyAuthenticated();
  }

  function bindTabs(tabLogin, tabRegister, loginForm, registerForm) {
    if (!tabLogin || !tabRegister || !loginForm || !registerForm) return;

    tabLogin.addEventListener('click', () => {
      tabLogin.classList.add('active');
      tabRegister.classList.remove('active');
      loginForm.classList.remove('hidden');
      registerForm.classList.add('hidden');
    });

    tabRegister.addEventListener('click', () => {
      tabRegister.classList.add('active');
      tabLogin.classList.remove('active');
      registerForm.classList.remove('hidden');
      loginForm.classList.add('hidden');
    });
  }

  function bindLoginForm(loginForm) {
    if (!loginForm) return;

    loginForm.addEventListener('submit', async (e) => {
      e.preventDefault();

      const email = document.getElementById('loginEmail')?.value?.trim() || '';
      const password = document.getElementById('loginPassword')?.value || '';

      if (!email || !password) {
        setMessage('Введите email и пароль', 'error');
        return;
      }

      if (!isValidEmail(email)) {
        setMessage('Пожалуйста, введите корректный адрес электронной почты', 'error');
        return;
      }

      const btn = document.getElementById('loginSubmit');

      if (btn) btn.disabled = true;
      setMessage('Выполняем вход...', 'info', true);

      try {
        await window.Api.loginUser({ email, password });

        setMessage('Вход успешен. Перенаправление...', 'success', false);

        setTimeout(() => {
          window.location.href = '/';
        }, 500);
      } catch (err) {
        handleAuthError(err, {
          invalid_request: 'Некорректные данные',
          invalid_credentials: 'Неверный email или пароль',
          unauthorized: 'Неверный email или пароль'
        });
      } finally {
        if (btn) btn.disabled = false;
      }
    });
  }

  function bindRegisterForm(registerForm) {
    if (!registerForm) return;

    registerForm.addEventListener('submit', async (e) => {
      e.preventDefault();

      const username = document.getElementById('regUsername')?.value?.trim() || '';
      const email = document.getElementById('regEmail')?.value?.trim() || '';
      const password = document.getElementById('regPassword')?.value || '';
      const confirm = document.getElementById('regConfirm')?.value || '';

      if (!username || !email || !password || !confirm) {
        setMessage('Заполните все поля', 'error');
        return;
      }

      if (!isValidEmail(email)) {
        setMessage('Пожалуйста, введите корректный адрес электронной почты', 'error');
        return;
      }

      if (password !== confirm) {
        setMessage('Пароли не совпадают', 'error');
        return;
      }

      const btn = document.getElementById('registerSubmit');

      if (btn) btn.disabled = true;
      setMessage('Отправка регистрации...', 'info', true);

      try {
        await window.Api.registerUser({ username, email, password });

        setMessage('Регистрация успешна. Пожалуйста, войдите в аккаунт.', 'success');

        const tabLogin = document.getElementById('tab-login');
        tabLogin?.click();
      } catch (err) {
        handleAuthError(err, {
          invalid_request: 'Некорректные данные',
          invalid_email: 'Некорректный email',
          invalid_password: 'Некорректный пароль',
          email_already_exists: 'Указанный email уже занят'
        });
      } finally {
        if (btn) btn.disabled = false;
      }
    });
  }

  function bindLogoutButton(logoutBtn) {
    if (!logoutBtn) return;

    logoutBtn.addEventListener('click', async () => {
      if (!confirm('Выйти из аккаунта?')) return;

      logoutBtn.disabled = true;
      setMessage('Выполняем выход...', 'info', true);

      try {
        await window.Api.logoutUser();

        setMessage('Вы вышли из аккаунта. Перенаправление на главную...', 'success');

        setTimeout(() => {
          window.location.href = '/';
        }, 600);
      } catch (err) {
        setMessage(err?.message || 'Ошибка выхода. Повторите попытку.', 'error');
      } finally {
        logoutBtn.disabled = false;
      }
    });
  }

  async function checkAlreadyAuthenticated() {
    try {
      const result = await window.Api.getMe();

      if (result?.authenticated) {
        renderAlreadyAuthenticated(result.user);
      }
    } catch (err) {
      console.warn('Не удалось проверить авторизацию:', err);
    }
  }

  function renderAlreadyAuthenticated(user) {
    const authCard = document.querySelector('.auth-card');
    if (!authCard) return;

    authCard.innerHTML = '';

    const header = document.createElement('div');
    header.className = 'auth-header';
    header.textContent = 'Вы уже авторизованы';

    const message = document.createElement('div');
    message.id = 'authMessage';
    message.textContent = user?.username
      ? `Вы уже вошли в аккаунт как ${user.username}.`
      : 'Вы уже вошли в аккаунт.';

    const actions = document.createElement('div');
    actions.className = 'auth-actions';

    const cabinetLink = document.createElement('a');
    cabinetLink.className = 'link-as-btn primary';
    cabinetLink.href = '/me';
    cabinetLink.textContent = 'Перейти в кабинет';

    const logoutBtn = document.createElement('button');
    logoutBtn.id = 'logoutBtn';
    logoutBtn.className = 'btn';
    logoutBtn.type = 'button';
    logoutBtn.textContent = 'Выйти';

    actions.appendChild(cabinetLink);
    actions.appendChild(logoutBtn);

    authCard.appendChild(header);
    authCard.appendChild(message);
    authCard.appendChild(actions);

    bindLogoutButton(logoutBtn);
  }

  function handleAuthError(err, messages = {}) {
    const message =
      messages[err?.code] ||
      err?.message ||
      'Ошибка доступа к серверу. Повторите попытку.';

    setMessage(message, 'error');
  }

  function isValidEmail(email) {
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
  }
if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    initAuthModule,

    setMessage,

    bindTabs,
    bindLoginForm,
    bindRegisterForm,
    bindLogoutButton,

    checkAlreadyAuthenticated,
    renderAlreadyAuthenticated,

    handleAuthError,
    isValidEmail
  };
}