const BATCH_SIZE = 100;
const RECONNECT_DELAYS = [1, 2, 5, 10, 30];
const MAX_RECONNECT_ATTEMPTS = 5;

let reconnectAttempt = 0;
let ws = null;
let expecting = null;
let city = document.body.dataset.initialCity || 'Москва';
let isAuthenticated = false;
let allItems = [];
let allCompanies = [];
let wsState = 'idle';
let manualRetryTimer = null;
let statusInterval = null;
let isConnectionClosedByServer = false;
let infiniteScrollObserver = null;
let pendingRequestCount = 0; 
let filtering = false; 
let filtersEnabled = false; 

const cityNameEl = document.getElementById('cityName');
const cityBtn = document.getElementById('cityBtn');
const cityModal = document.getElementById('cityModal');
const cityListEl = document.getElementById('cityList');
const citySearch = document.getElementById('citySearch');
const cityOk = document.getElementById('cityOk');
const cityCancel = document.getElementById('cityCancel');
const authBtn = document.getElementById('authButton');
const cardsWrapper = document.getElementById('cardsWrapper');
const statusBar = document.getElementById('statusBar');
const companyListEl = document.getElementById('companyList');
const companyToggleWrap = document.getElementById('companyToggleWrap');
const priceInput = document.getElementById('priceInput');
const capacityInput = document.getElementById('capacityInput');
const applyFiltersBtn = document.getElementById('applyFilters');
const clearFiltersBtn = document.getElementById('clearFilters');
const brandLink = document.getElementById('brand');

if (cityNameEl) cityNameEl.textContent = city;

if (authBtn) {
  authBtn.addEventListener('click', (e) => {
    e.stopPropagation();
    const isAuth = authBtn.dataset && authBtn.dataset.auth === '1';
    if (isAuth) {
      navigate('/me');
    } else {
      navigate('/auth');
    }
  });
}
function navigate(url) {
  window.location.href = url;
}

window.addEventListener('error', (ev) => {
  console.error('Unhandled error:', ev.error || ev.message, ev);
  try {
    showCenteredMessage('Произошла ошибка. Обновите страницу или попробуйте позже.', false);
  } catch (e) {}
});

function setFiltersEnabled(enabled) {
  filtersEnabled = !!enabled;
  if (applyFiltersBtn) applyFiltersBtn.disabled = !filtersEnabled;
  if (clearFiltersBtn) clearFiltersBtn.disabled = !filtersEnabled;
  if (!companyListEl) return;
  companyListEl.querySelectorAll('input[type="checkbox"]').forEach(cb => cb.disabled = !filtersEnabled);
}
function resetState() {
  reconnectAttempt = 0;
  ws = null;
  expecting = null;
  isAuthenticated = false;
  allItems = [];
  allCompanies = [];
  wsState = 'idle';
  manualRetryTimer = null;
  statusInterval = null;
  isConnectionClosedByServer = false;
  infiniteScrollObserver = null;
  pendingRequestCount = 0;
  filtering = false;
  filtersEnabled = false;
}
function openCityModal() {
  if (!cityModal) return;
  try { cityModal.showModal(); } catch (err) {}
  citySearch.value = '';
  filterCityList('');
  hideCityError();
  cityListEl.querySelectorAll('.city-item').forEach(item => {
    if (item.textContent.trim() === city) {
      item.classList.add('selected');
      citySearch.value = city;
    } else {
      item.classList.remove('selected');
    }
  });
}
function closeCityModal() {
  if (!cityModal) return;
  try { cityModal.close(); } catch (e) {}
}
function filterCityList(searchText) {
  if (!cityListEl) return;
  const items = cityListEl.querySelectorAll('.city-item');
  let hasVisible = false;
  items.forEach(item => {
    const cityName = item.textContent.toLowerCase();
    if (cityName.includes(searchText.toLowerCase())) {
      item.style.display = 'block';
      hasVisible = true;
    } else {
      item.style.display = 'none';
      item.classList.remove('selected');
    }
  });
  if (!hasVisible) {
    const noResults = document.createElement('li');
    noResults.className = 'city-item';
    noResults.textContent = 'Город не найден';
    noResults.style.color = 'var(--muted)';
    noResults.style.fontStyle = 'italic';
    cityListEl.appendChild(noResults);
  } else {
    const noResults = cityListEl.querySelector('.city-item[style*="italic"]');
    if (noResults) noResults.remove();
  }
  const selected = cityListEl.querySelector('.city-item.selected');
  if (selected && selected.style.display === 'none') selected.classList.remove('selected');
}

const availableCities = [
  'Москва', 'Санкт-Петербург', 'Казань', 'Екатеринбург', 'Новосибирск', 'Нижний Новгород'
];

function setCity(newCity) {
  const normalizedCity = newCity.trim();
  if (!availableCities.includes(normalizedCity)) {
    showCityError('Такого города не существует');
    return;
  }
  if (normalizedCity && normalizedCity !== city) {
    city = normalizedCity;
    if (cityNameEl) cityNameEl.textContent = city;
    document.body.dataset.initialCity = city;

    allItems = [];
    allCompanies = [];
    isConnectionClosedByServer = false;
    pendingRequestCount = 0;
    filtering = false;

    setFiltersEnabled(false);

    if (ws) {
      try { ws.close(); } catch (e) {}
      ws = null;
    }
    if (manualRetryTimer) { clearTimeout(manualRetryTimer); manualRetryTimer = null; }
    if (statusInterval) { clearInterval(statusInterval); statusInterval = null; }

    showCenteredMessage(`Загружаем данные для ${city}...`, false);
    connectWebSocket();
  }
}

function showCityError(message) {
  hideCityError();
  const errorElement = document.createElement('div');
  errorElement.id = 'cityError';
  errorElement.style.color = '#dc2626';
  errorElement.style.fontSize = '14px';
  errorElement.style.marginTop = '8px';
  errorElement.style.textAlign = 'center';
  errorElement.textContent = message;
  if (cityListEl && cityListEl.parentNode) {
    cityListEl.parentNode.insertBefore(errorElement, cityListEl.nextSibling);
    setTimeout(() => hideCityError(), 3000);
  }
}
function hideCityError() {
  const existingError = document.getElementById('cityError');
  if (existingError) existingError.remove();
}

if (cityBtn) cityBtn.addEventListener('click', () => openCityModal());
if (cityCancel) cityCancel.addEventListener('click', () => closeCityModal());
if (cityOk) cityOk.addEventListener('click', () => {
  const selected = cityListEl ? cityListEl.querySelector('.city-item.selected') : null;
  const searchValue = citySearch ? citySearch.value.trim() : '';
  if (selected) {
    setCity(selected.textContent.trim());
    closeCityModal();
  } else if (searchValue) {
    setCity(searchValue);
    closeCityModal();
  } else showCityError('Выберите город из списка или введите название');
});
if (citySearch) {
  citySearch.addEventListener('input', () => {
    filterCityList(citySearch.value.trim());
    if (citySearch.value.trim() && cityListEl) cityListEl.querySelectorAll('.city-item').forEach(item => item.classList.remove('selected'));
  });
  citySearch.addEventListener('keyup', (e) => {
    if (e.key === 'Enter') {
      const searchValue = citySearch.value.trim();
      if (searchValue) setCity(searchValue);
      else showCityError('Введите название города');
    }
  });
}
if (cityListEl) {
  cityListEl.addEventListener('click', (e) => {
    const item = e.target.closest('.city-item');
    if (!item) return;
    if (item.style.fontStyle) return; 
    cityListEl.querySelectorAll('.city-item').forEach(n => n.classList.remove('selected'));
    item.classList.add('selected');
    if (citySearch) citySearch.value = item.textContent.trim();
  });
}
if (cityModal) {
  cityModal.addEventListener('click', (e) => {
    const dialogDimensions = cityModal.getBoundingClientRect();
    if (e.clientX < dialogDimensions.left || e.clientX > dialogDimensions.right || e.clientY < dialogDimensions.top || e.clientY > dialogDimensions.bottom) {
      closeCityModal();
    }
  });
  cityModal.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') closeCityModal();
  });
}

if (brandLink) brandLink.addEventListener('click', (e) => {
  e.preventDefault();
  window.location.href = '/';
});

if (applyFiltersBtn) applyFiltersBtn.addEventListener('click', () => applyFilters(true));
if (clearFiltersBtn) clearFiltersBtn.addEventListener('click', () => {
  if (priceInput) priceInput.value = '';
  if (capacityInput) capacityInput.value = '';
  if (companyListEl) companyListEl.querySelectorAll('input[type="checkbox"]').forEach(cb => cb.checked = false);
  applyFilters(true);
});

if (priceInput) {
  priceInput.addEventListener('input', (e) => {
    e.target.value = e.target.value.replace(/[^\d]/g, '');
    if (e.target.value.length > 1 && e.target.value[0] === '0') {
      e.target.value = e.target.value.substring(1);
    }
  });
  priceInput.addEventListener('blur', () => {
    let price = parseInt(priceInput.value) || 0;
    if (price < 0) price = 0;
    priceInput.value = price > 0 ? price : '';
  });
  priceInput.addEventListener('keydown', (e) => {
    if ((e.key >= '0' && e.key <= '9') || e.key === 'Backspace' || e.key === 'Delete' || e.key === 'Tab' || e.key === 'ArrowLeft' || e.key === 'ArrowRight' || e.key === 'ArrowUp' || e.key === 'ArrowDown') {
      return;
    }
    e.preventDefault();
  });
}

if (capacityInput) {
  capacityInput.addEventListener("input", () => {
    capacityInput.value = capacityInput.value.replace(/\D/g, "");
  });
}

function ensureSentinel(enable = true) {
  try {
    if (!cardsWrapper) return;
    const oldSentinel = cardsWrapper.querySelector('#sentinel');
    if (oldSentinel) oldSentinel.remove();

    if (!enable) {
      if (infiniteScrollObserver) {
        infiniteScrollObserver.disconnect();
        infiniteScrollObserver = null;
      }
      return;
    }

    const sentinel = document.createElement('div');
    sentinel.id = 'sentinel';
    sentinel.style.height = '1px';
    cardsWrapper.appendChild(sentinel);

    if (infiniteScrollObserver) infiniteScrollObserver.disconnect();
    infiniteScrollObserver = new IntersectionObserver(entries => {
      for (const entry of entries) {
        if (entry.isIntersecting) {
          requestMoreData();
        }
      }
    }, { root: null, rootMargin: '0px', threshold: 0.1 });

    infiniteScrollObserver.observe(sentinel);
  } catch (err) {
    console.error('ensureSentinel error:', err);
  }
}

function getWsUrl() {
  const scheme = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${scheme}//${window.location.host}/ws/home`;
}

function connectWebSocket() {
  if (isConnectionClosedByServer) {
    showCenteredMessage('В этом городе нет помещений для бронирования', false);
    return;
  }
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
    return;
  }
  const url = getWsUrl();
  showStatusBar('Подключение к серверу...');
  try {
    ws = new WebSocket(url);
  } catch (err) {
    onWsFailure();
    return;
  }

  ws.addEventListener('open', () => {
    reconnectAttempt = 0;
    showStatusBar('Соединение установлено. Проверяю наличие помещений...');
    try {  
      ws.send(city);
    } catch (err) {
      console.warn(err);
    }
    if (pendingRequestCount > 0) {
      try {
        ws.send(String(pendingRequestCount));
        showStatusBar('Запрашиваем данные...');
      } catch (err) { console.warn(err); }
      pendingRequestCount = 0;
    }
  });

  ws.addEventListener('message', ev => handleWsMessage(ev.data));

  ws.addEventListener('close', (event) => {
    if (event.code !== 1000) { 
      isConnectionClosedByServer = true;
    }
    onWsFailure();
  });

  ws.addEventListener('error', () => {
    onWsFailure();
  });
}

function onWsFailure() {
  if (isConnectionClosedByServer) {
    if (!allItems.length) {
      showCenteredMessage('В этом городе нет помещений для бронирования', false);
    } else {
      hideStatusBar();
    }
    return;
  }

  if (!allItems.length) {
    showCenteredMessage(`Не удалось подключиться к серверу. Попробуем через ${RECONNECT_DELAYS[0]}с...`, true);
  } else {
    hideStatusBar();
  }
  scheduleReconnect();
}

function scheduleReconnect() {
  if (manualRetryTimer || isConnectionClosedByServer) return;
  const delayIndex = Math.min(reconnectAttempt, MAX_RECONNECT_ATTEMPTS - 1);
  const delay = RECONNECT_DELAYS[delayIndex];
  let remaining = delay;

  if (!allItems.length) {
    showCenteredMessage(`Не удалось подключиться к серверу. Попытка через ${remaining}с...`, true);
    statusInterval && clearInterval(statusInterval);
    statusInterval = setInterval(() => {
      remaining--;
      if (remaining <= 0) {
        clearInterval(statusInterval);
        statusInterval = null;
      } else {
        showCenteredMessage(`Не удалось подключиться к серверу. Попытка через ${remaining}с...`, true);
      }
    }, 1000);
  } else {
    hideStatusBar();
  }

  manualRetryTimer = setTimeout(() => {
    manualRetryTimer = null;
    reconnectAttempt++;
    module.exports.connectWebSocket();
  }, delay * 1000);
}

function showStatusBar(text) {
  if (statusBar) {
    statusBar.hidden = false;
    statusBar.textContent = text;
  }
}
function hideStatusBar() {
  if (statusBar) {
    statusBar.hidden = true;
    statusBar.textContent = '';
  }
}

function handleWsMessage(raw) {
  try {
    const data = JSON.parse(raw);

    if (data === null) {
      isConnectionClosedByServer = true;
      if (ws) ws.close();
      showCenteredMessage('В этом городе нет помещений для бронирования', false);
      return;
    }

    if (allCompanies.length === 0 && Array.isArray(data)) {
      allCompanies = data;
      refreshFilterValues();
      setFiltersEnabled(true);

      if (ws && ws.readyState === WebSocket.OPEN) {
        try {
          ws.send(String(BATCH_SIZE));
          showStatusBar('Запрашиваем данные...');
        } catch (err) {
          console.warn(err);
        }
      } else {
        pendingRequestCount += BATCH_SIZE;
      }
      return;
    }
    if (Array.isArray(data)) {
      if (data.length === 0) {
        isConnectionClosedByServer = true;
        if (ws) ws.close();
        if (!allItems.length) {
          showCenteredMessage('В этом городе нет помещений для бронирования', false);
        }
      } else {
        addItems(data);
        hideStatusBar();
      }
    }
  } catch (err) {
    console.warn('Parse error', err);
  }
}

function requestMoreData() {

  if (filtering) return;

  if (ws && ws.readyState === WebSocket.OPEN && !isConnectionClosedByServer) {
    try {
      ws.send(String(BATCH_SIZE));
      showStatusBar('Запрашиваем данные...');
    } catch (err) {
      console.warn(err);
    }
  } else {
    pendingRequestCount += BATCH_SIZE;
  }
}

function addItems(items) {
  allItems = allItems.concat(items);
  filtering = false;
  renderCards(allItems, true);
  refreshFilterValues();
  applyFilters(false);
}

function renderCards(items, attachSentinel = true) {
  try {
    if (!cardsWrapper) return;
    cardsWrapper.innerHTML = '';

    if (!items || items.length === 0) {
      const empty = document.createElement('div');
      empty.className = 'empty';
      empty.textContent = 'Ничего не найдено по выбранным фильтрам';
      cardsWrapper.appendChild(empty);
      ensureSentinel(attachSentinel);
      return;
    }

    for (const it of items) {
      try {
        const btn = document.createElement('button');
        btn.className = 'card';
        btn.type = 'button';
        btn.setAttribute('aria-label', `${it.title || ''} — ${it.company || ''}`);
        btn.addEventListener('click', () => {
          if (it.id) {
            navigate(`/room/${encodeURIComponent(it.id)}`);
          } else {
            console.warn('id отсутствует у карточки', it);
          }
        });

        const img = document.createElement('img');
        img.className = 'card__img';
        img.alt = it.title || '';
        img.src = it.image || "data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='600' height='400'></svg>";

        const body = document.createElement('div');
        body.className = 'card__body';

        const title = document.createElement('div');
        title.className = 'card__title';
        title.textContent = it.title || '';

        const metaTop = document.createElement('div');
        const company = document.createElement('div');
        company.className = 'card__company';
        company.textContent = it.company || '';
        const address = document.createElement('div');
        address.className = 'card__address';
        address.textContent = it.address || '';
        metaTop.appendChild(company);
        metaTop.appendChild(address);

        const row = document.createElement('div');
        row.className = 'card__row';
        const capacity = document.createElement('div');
        capacity.className = 'capacity';
        const capIcon = document.createElement('span');
        capIcon.className = 'cap-icon';
        capIcon.textContent = '👥';
        const capText = document.createElement('span');
        capText.textContent = `до ${it.capacity || 0}`;
        capacity.appendChild(capIcon);
        capacity.appendChild(capText);

        const priceBadge = document.createElement('div');
        priceBadge.className = 'price-badge';
        priceBadge.textContent = `${it.price || 0} ₽/ч`;

        row.appendChild(capacity);
        row.appendChild(priceBadge);

        body.appendChild(title);
        body.appendChild(metaTop);
        body.appendChild(row);

        btn.appendChild(img);
        btn.appendChild(body);
        cardsWrapper.appendChild(btn);
      } catch (innerErr) {
        console.warn('render card error', innerErr);
      }
    }

    ensureSentinel(attachSentinel);
  } catch (err) {
    console.error('renderCards error', err);
    showCenteredMessage('Произошла ошибка при отображении карточек', false);
  }
}

function showCenteredMessage(text, showRetry) {
  if (!cardsWrapper) return;
  cardsWrapper.innerHTML = '';
  const container = document.createElement('div');
  container.className = 'center-message';
  const p = document.createElement('div');
  p.textContent = text;
  container.appendChild(p);

  if (showRetry) {
    const btnGroup = document.createElement('div');
    btnGroup.style.marginTop = '6px';
    const retryBtn = document.createElement('button');
    retryBtn.className = 'btn';
    retryBtn.textContent = 'Попробовать сейчас';
    retryBtn.addEventListener('click', () => {
      if (manualRetryTimer) { clearTimeout(manualRetryTimer); manualRetryTimer = null; }
      if (statusInterval) { clearInterval(statusInterval); statusInterval = null; }
      showCenteredMessage('Подключаемся...', false);
      reconnectAttempt = Math.max(0, reconnectAttempt - 1);
      module.exports.connectWebSocket();
    });
    btnGroup.appendChild(retryBtn);
    container.appendChild(btnGroup);
  }

  cardsWrapper.appendChild(container);
}

function refreshFilterValues() {
  try {
    if (!companyListEl) return;
    companyListEl.innerHTML = '';
    if (allCompanies.length === 0) {
      companyListEl.innerHTML = '<div class="placeholder">Нет данных</div>';
      companyToggleWrap.innerHTML = '';
    } else {
      allCompanies.forEach((c, idx) => {
        const id = `company_${idx}_${Math.random().toString(36).slice(2,6)}`;
        const wrapper = document.createElement('label');
        wrapper.style.display = 'flex';
        wrapper.style.alignItems = 'center';
        wrapper.style.gap = '8px';
        wrapper.style.cursor = 'pointer';
        const cb = document.createElement('input');
        cb.type = 'checkbox';
        cb.value = c;
        cb.id = id;
        cb.disabled = !filtersEnabled;
        const span = document.createElement('span');
        span.textContent = c;
        wrapper.appendChild(cb);
        wrapper.appendChild(span);
        companyListEl.appendChild(wrapper);
      });

      if (allCompanies.length > 6) {
        companyListEl.style.maxHeight = '160px';
        companyToggleWrap.innerHTML = '';
        const toggleBtn = document.createElement('button');
        toggleBtn.className = 'btn';
        toggleBtn.textContent = 'Показать всё';
        let expanded = false;
        toggleBtn.addEventListener('click', () => {
          expanded = !expanded;
          if (expanded) {
            companyListEl.style.maxHeight = '360px';
            toggleBtn.textContent = 'Свернуть';
          } else {
            companyListEl.style.maxHeight = '160px';
            toggleBtn.textContent = 'Показать всё';
          }
        });
        companyToggleWrap.appendChild(toggleBtn);
      } else {
        companyToggleWrap.innerHTML = '';
        companyListEl.style.maxHeight = 'none';
      }
    }
  } catch (err) {
    console.error('refreshFilterValues error', err);
  }
}

function applyFilters(userTriggered = false) {
  try {
    if (allItems.length === 0) {
      if (userTriggered) {
        renderCards([], false);
      }
      return;
    }

    let filtered = [...allItems];
    const priceValue = priceInput ? Number(priceInput.value) : 0;
    if (priceValue > 0) {
      filtered = filtered.filter(it => it.price <= priceValue);
    }
    const capacityValue = capacityInput ? Number(capacityInput.value) : 0;
    if (capacityValue > 0) {
      filtered = filtered.filter(it => it.capacity >= capacityValue);
    }
    const selectedCompanies = companyListEl ? Array.from(companyListEl.querySelectorAll('input[type="checkbox"]:checked'))
      .map(cb => cb.value.trim())
      .filter(company => company !== '') : [];
    if (selectedCompanies.length > 0) {
      filtered = filtered.filter(it => selectedCompanies.includes(it.company));
    }

    const filtersActive = (priceValue > 0) || (capacityValue > 0) || (selectedCompanies.length > 0);
    filtering = userTriggered && filtersActive;
    if (userTriggered) {
      renderCards(filtered, !filtering);
      if (typeof updateResultsCounter === 'function') updateResultsCounter(filtered.length);
    } else {
      return filtered;
    }
  } catch (err) {
    console.error('applyFilters error', err);
    showCenteredMessage('Ошибка при применении фильтров', false);
  }
}

ensureSentinel();
connectWebSocket();

window.debug_state = function() {
  return {city, wsState, allItemsCount: allItems.length, wsReadyState: ws ? ws.readyState : 'no-ws', reconnectAttempt, filtering, pendingRequestCount, filtersEnabled};
};
if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    filterCityList, showCityError, hideCityError,
    setCity, setFiltersEnabled, applyFilters,
    refreshFilterValues, renderCards, showCenteredMessage,
    handleWsMessage, requestMoreData, addItems,
    getWsUrl, connectWebSocket, scheduleReconnect,
    navigate, resetState
  };
}
