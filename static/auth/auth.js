
function setMessage(text, type = 'info', sticky = false) {
  const el = document.getElementById('authMessage');
  if (!el) return;
  el.textContent = text || '';
  if (type === 'error') el.style.color = '#dc2626';
  else if (type === 'success') el.style.color = '#065f46';
  else el.style.color = 'var(--muted)';
  if (!sticky) {
    clearTimeout(el._hideTimer);
    el._hideTimer = setTimeout(() => {
      el.textContent = '';
      el.style.color = 'var(--muted)';
    }, 5000);
  }
}


async function postJson(url, payload) {
  return fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'same-origin',
    body: JSON.stringify(payload)
  });
}

function initAuthModule(){
  const alreadyAuth = !!(window.__USER__ && window.__USER__.auth);
  const tabLogin = document.getElementById('tab-login');
  const tabRegister = document.getElementById('tab-register');
  const loginForm = document.getElementById('loginForm');
  const registerForm = document.getElementById('registerForm');
  const logoutBtn = document.getElementById('logoutBtn');
  if (alreadyAuth) {
    return;
  }


  if (tabLogin && tabRegister && loginForm && registerForm) {
    tabLogin.addEventListener('click', () => {
      tabLogin.classList.add('active'); tabRegister.classList.remove('active');
      loginForm.classList.remove('hidden'); registerForm.classList.add('hidden');
    });
    tabRegister.addEventListener('click', () => {
      tabRegister.classList.add('active'); tabLogin.classList.remove('active');
      registerForm.classList.remove('hidden'); loginForm.classList.add('hidden');
    });
  }

  if (registerForm) {
    registerForm.addEventListener('submit', async (e) => {
      e.preventDefault();
      const username = (document.getElementById('regUsername')||{}).value?.trim() || '';
      const email = (document.getElementById('regEmail')||{}).value?.trim() || '';
      const password = (document.getElementById('regPassword')||{}).value || '';
      const confirm = (document.getElementById('regConfirm')||{}).value || '';
      const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
 

      if (!username || !email || !password || !confirm) { setMessage('Заполните все поля', 'error'); return; }
      if (!emailRegex.test(email)) {
        setMessage('Пожалуйста, введите корректный адрес электронной почты (например, name@mail.com)', 'error');
        return;
      }
      if (password !== confirm) { setMessage('Пароли не совпадают', 'error'); return; }

      const btn = document.getElementById('registerSubmit');
      btn.disabled = true;
      setMessage('Отправка регистрации...', 'info', true);

      try {
        const res = await postJson('/api/register', { username, email, password });
        if (res.ok) {
          setMessage('Регистрация успешна. Пожалуйста, войдите в аккаунт.', 'success');
          tabLogin?.click();
        } else {
          switch (res.status) {
            case 400:
              setMessage('Некорректные данные', 'error');
              break;
            case 409:
              setMessage('Указанный email уже занят', 'error');
              break;
            case 500:
            default:
              setMessage('Ошибка доступа к серверу. Повторите попытку.', 'error');
              break;
          }
        }
      } catch (err) {
        setMessage('Ошибка соединения', 'error');
      } finally {
        btn.disabled = false;
      }
    });
  }

  if (loginForm) {
    loginForm.addEventListener('submit', async (e) => {
      e.preventDefault();
      const email = (document.getElementById('loginEmail')||{}).value?.trim() || '';
      const password = (document.getElementById('loginPassword')||{}).value || '';

      if (!email || !password) { setMessage('Введите email и пароль', 'error'); return; }
      const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
      if (!emailRegex.test(email)) {
        setMessage('Пожалуйста, введите корректный адрес электронной почты (например, name@mail.com)', 'error');
        return;
      }

      const btn = document.getElementById('loginSubmit');
      btn.disabled = true;
      setMessage('Выполняем вход...', 'info', true);

      try {
        const res = await postJson('/api/login', { email, password });
        if (res.ok) {
          setMessage('Вход успешен. Перенаправление...', 'success', false);
          setTimeout(() => { window.location.href = '/'; }, 500);
        } else {
          switch (res.status) {
            case 400:
              setMessage('Некорректные данные', 'error');
              break;
            case 401:
              setMessage('Неверный email или пароль', 'error');
              break;
            case 500:
            default:
              setMessage('Ошибка доступа к серверу. Повторите попытку.', 'error');
              break;
          }
        }
      } catch (err) {
        setMessage('Ошибка соединения', 'error');
      } finally {
        btn.disabled = false;
      }
    });

  }

  if (logoutBtn) {
    logoutBtn.addEventListener('click', async () => {
      if (!confirm('Выйти из аккаунта?')) return;
      logoutBtn.disabled = true;
      setMessage('Выполняем выход...', 'info', true);
      try {
        const res = await fetch('/api/logout', { method: 'POST', credentials: 'same-origin' });
        if (res.ok) {
          setMessage('Вы вышли из аккаунта. Перенаправление на главную...', 'success');
          setTimeout(() => { window.location.href = '/'; }, 600);
        } else {
          switch (res.status) {
            case 400:
              setMessage('Сессия отсутствует или истекла', 'error');
              break;
            case 500:
            default:
              setMessage('Ошибка доступа к серверу. Повторите попытку.', 'error');
              break;
          }
        }
      } catch (err) {
        setMessage('Ошибка соединения', 'error');
      } finally {
        logoutBtn.disabled = false;
      }
    });
  }
}
document.addEventListener('DOMContentLoaded', initAuthModule);

if (typeof module !== 'undefined' && module.exports) {
  module.exports = { setMessage, postJson, initAuthModule };
}
