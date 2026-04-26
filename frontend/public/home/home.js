(function () {
  'use strict';

  const BATCH_SIZE = 100;

  const availableCities = [
    'Москва',
    'Санкт-Петербург',
    'Казань',
    'Екатеринбург',
    'Новосибирск',
    'Нижний Новгород'
  ];

  let city = document.body.dataset.initialCity || 'Москва';
  let isAuthenticated = false;
  let currentUserRole = null;
  let allItems = [];
  let allCompanies = [];

  let lastAfterId = 0;
  let hasMore = true;
  let isLoading = false;
  let filtering = false;
  let filtersEnabled = false;
  let infiniteScrollObserver = null;

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

  document.addEventListener('DOMContentLoaded', init);

  async function init() {
    if (cityNameEl) {
      cityNameEl.textContent = city;
    }

    bindEvents();
    setFiltersEnabled(false);

    await loadCurrentUser();
    await loadInitialData();
  }

  function bindEvents() {
      if (authBtn) {
        authBtn.addEventListener('click', (e) => {
          e.stopPropagation();

          if (!isAuthenticated) {
            navigate('/auth');
            return;
          }

          if (currentUserRole === 'superuser') {
            navigate('/superuser');
            return;
          }

          if (currentUserRole === 'admin') {
            navigate('/admin');
            return;
          }

          navigate('/me');
        });
      }

    if (brandLink) {
      brandLink.addEventListener('click', (e) => {
        e.preventDefault();
        navigate('/');
      });
    }

    if (cityBtn) {
      cityBtn.addEventListener('click', openCityModal);
    }

    if (cityCancel) {
      cityCancel.addEventListener('click', closeCityModal);
    }

    if (cityOk) {
      cityOk.addEventListener('click', () => {
        const selected = cityListEl ? cityListEl.querySelector('.city-item.selected') : null;
        const searchValue = citySearch ? citySearch.value.trim() : '';

        if (selected) {
          setCity(selected.textContent.trim());
          closeCityModal();
          return;
        }

        if (searchValue) {
          setCity(searchValue);
          closeCityModal();
          return;
        }

        showCityError('Выберите город из списка или введите название');
      });
    }

    if (citySearch) {
      citySearch.addEventListener('input', () => {
        filterCityList(citySearch.value.trim());

        if (citySearch.value.trim() && cityListEl) {
          cityListEl
            .querySelectorAll('.city-item')
            .forEach(item => item.classList.remove('selected'));
        }
      });

      citySearch.addEventListener('keyup', (e) => {
        if (e.key === 'Enter') {
          const searchValue = citySearch.value.trim();

          if (searchValue) {
            setCity(searchValue);
            closeCityModal();
          } else {
            showCityError('Введите название города');
          }
        }
      });
    }

    if (cityListEl) {
      cityListEl.addEventListener('click', (e) => {
        const item = e.target.closest('.city-item');
        if (!item) return;
        if (item.style.fontStyle) return;

        cityListEl
          .querySelectorAll('.city-item')
          .forEach(n => n.classList.remove('selected'));

        item.classList.add('selected');

        if (citySearch) {
          citySearch.value = item.textContent.trim();
        }
      });
    }

    if (cityModal) {
      cityModal.addEventListener('click', (e) => {
        const rect = cityModal.getBoundingClientRect();

        if (
          e.clientX < rect.left ||
          e.clientX > rect.right ||
          e.clientY < rect.top ||
          e.clientY > rect.bottom
        ) {
          closeCityModal();
        }
      });

      cityModal.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
          closeCityModal();
        }
      });
    }

    if (applyFiltersBtn) {
      applyFiltersBtn.addEventListener('click', () => applyFilters(true));
    }

    if (clearFiltersBtn) {
      clearFiltersBtn.addEventListener('click', () => {
        if (priceInput) priceInput.value = '';
        if (capacityInput) capacityInput.value = '';

        if (companyListEl) {
          companyListEl
            .querySelectorAll('input[type="checkbox"]')
            .forEach(cb => cb.checked = false);
        }

        applyFilters(true);
      });
    }

    if (priceInput) {
      priceInput.addEventListener('input', (e) => {
        e.target.value = e.target.value.replace(/[^\d]/g, '');

        if (e.target.value.length > 1 && e.target.value[0] === '0') {
          e.target.value = e.target.value.substring(1);
        }
      });

      priceInput.addEventListener('blur', () => {
        let price = parseInt(priceInput.value, 10) || 0;
        if (price < 0) price = 0;
        priceInput.value = price > 0 ? String(price) : '';
      });

      priceInput.addEventListener('keydown', (e) => {
        const allowed = [
          'Backspace',
          'Delete',
          'Tab',
          'ArrowLeft',
          'ArrowRight',
          'ArrowUp',
          'ArrowDown'
        ];

        if ((e.key >= '0' && e.key <= '9') || allowed.includes(e.key)) {
          return;
        }

        e.preventDefault();
      });
    }

    if (capacityInput) {
      capacityInput.addEventListener('input', () => {
        capacityInput.value = capacityInput.value.replace(/\D/g, '');
      });
    }

    window.addEventListener('error', (ev) => {
      console.error('Unhandled error:', ev.error || ev.message, ev);
      showCenteredMessage('Произошла ошибка. Обновите страницу или попробуйте позже.', false);
    });
  }

  async function loadCurrentUser() {
    try {
        const result = await window.Api.getMe();

        isAuthenticated = !!result?.authenticated;
        currentUserRole = result?.user?.role || null;

        if (authBtn) {
          authBtn.dataset.auth = isAuthenticated ? '1' : '0';
          authBtn.dataset.role = currentUserRole || '';

          if (!isAuthenticated) {
            authBtn.textContent = 'Войти';
          } else if (currentUserRole === 'superuser') {
            authBtn.textContent = 'Панель платформы';
          } else if (currentUserRole === 'admin') {
            authBtn.textContent = 'Админ-панель';
          } else {
            authBtn.textContent = 'Кабинет';
          }

          if (result?.user?.username) {
            authBtn.dataset.username = result.user.username;
          }
        }
    } catch (err) {
      console.warn('Не удалось получить текущего пользователя:', err);
      isAuthenticated = false;
      currentUserRole = null;

      if (authBtn) {
        authBtn.dataset.auth = '0';
        authBtn.dataset.role = '';
        authBtn.textContent = 'Войти';
      }
    }
  }

  async function loadInitialData() {
    resetCatalogState();

    showCenteredMessage(`Загружаем данные для ${city}...`, false);
    showStatusBar('Загрузка фильтров...');

    try {
      await loadFilters();
      await loadMoreRooms(true);
    } catch (err) {
      console.error(err);
      showCenteredMessage('Не удалось загрузить помещения. Попробуйте позже.', true);
    } finally {
      hideStatusBar();
    }
  }

  async function loadFilters() {
    allCompanies = [];

    try {
      const data = await window.Api.getRoomFilters(city);
      allCompanies = Array.isArray(data?.companies) ? data.companies : [];
    } catch (err) {
      console.warn('Фильтры не получены, компании будут собраны из карточек:', err);
      allCompanies = [];
    }

    refreshFilterValues();
    setFiltersEnabled(true);
  }

  async function loadMoreRooms(reset = false) {
    if (isLoading) return;
    if (!hasMore && !reset) return;

    isLoading = true;
    showStatusBar('Загружаем помещения...');

    if (reset) {
      allItems = [];
      lastAfterId = 0;
      hasMore = true;
    }

    try {
      const data = await window.Api.getRooms({
        city,
        limit: BATCH_SIZE,
        after_id: lastAfterId || 0
      });

      const items = Array.isArray(data?.items)
        ? data.items
        : Array.isArray(data)
          ? data
          : [];

      const pagination = data?.pagination || {};

      if (items.length > 0) {
        addItems(items);
      }

      if (pagination.next_after_id !== undefined && pagination.next_after_id !== null) {
        lastAfterId = pagination.next_after_id;
      } else if (items.length > 0) {
        lastAfterId = items[items.length - 1].id || lastAfterId;
      }

      if (typeof pagination.has_more === 'boolean') {
        hasMore = pagination.has_more;
      } else {
        hasMore = items.length === BATCH_SIZE;
      }

      if (allItems.length === 0) {
        showCenteredMessage('В этом городе нет помещений для бронирования', false);
        ensureSentinel(false);
        return;
      }

      hideStatusBar();
      ensureSentinel(!filtering && hasMore);
    } catch (err) {
      console.error(err);

      if (allItems.length === 0) {
        showCenteredMessage('Не удалось загрузить помещения. Попробуйте позже.', true);
      } else {
        hideStatusBar();
      }
    } finally {
      isLoading = false;
    }
  }

  function resetCatalogState() {
    allItems = [];
    allCompanies = [];
    lastAfterId = 0;
    hasMore = true;
    isLoading = false;
    filtering = false;

    if (companyListEl) {
      companyListEl.innerHTML = '<div class="placeholder">Загрузка...</div>';
    }

    if (companyToggleWrap) {
      companyToggleWrap.innerHTML = '';
    }

    if (infiniteScrollObserver) {
      infiniteScrollObserver.disconnect();
      infiniteScrollObserver = null;
    }
  }

  function addItems(items) {
    allItems = allItems.concat(items);

    if (allCompanies.length === 0) {
      allCompanies = [...new Set(allItems.map(it => it.company).filter(Boolean))];
      refreshFilterValues();
    }

    const filtered = applyFilters(false);
    renderCards(filtered || allItems, !filtering && hasMore);
  }

  function setCity(newCity) {
    const normalizedCity = newCity.trim();

    if (!availableCities.includes(normalizedCity)) {
      showCityError('Такого города не существует');
      return;
    }

    if (normalizedCity && normalizedCity !== city) {
      city = normalizedCity;

      if (cityNameEl) {
        cityNameEl.textContent = city;
      }

      document.body.dataset.initialCity = city;

      if (priceInput) priceInput.value = '';
      if (capacityInput) capacityInput.value = '';

      loadInitialData();
    }
  }

  function openCityModal() {
    if (!cityModal) return;

    try {
      cityModal.showModal();
    } catch (err) {
      console.warn(err);
    }

    if (citySearch) {
      citySearch.value = '';
    }

    filterCityList('');
    hideCityError();

    if (!cityListEl) return;

    cityListEl.querySelectorAll('.city-item').forEach(item => {
      if (item.textContent.trim() === city) {
        item.classList.add('selected');
        if (citySearch) citySearch.value = city;
      } else {
        item.classList.remove('selected');
      }
    });
  }

  function closeCityModal() {
    if (!cityModal) return;

    try {
      cityModal.close();
    } catch (err) {
      console.warn(err);
    }
  }

  function filterCityList(searchText) {
    if (!cityListEl) return;

    const oldNoResults = cityListEl.querySelector('[data-no-results="1"]');
    if (oldNoResults) oldNoResults.remove();

    const items = cityListEl.querySelectorAll('.city-item:not([data-no-results="1"])');
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
      noResults.dataset.noResults = '1';
      noResults.textContent = 'Город не найден';
      noResults.style.color = 'var(--muted)';
      noResults.style.fontStyle = 'italic';
      cityListEl.appendChild(noResults);
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
      setTimeout(hideCityError, 3000);
    }
  }

  function hideCityError() {
    const existingError = document.getElementById('cityError');
    if (existingError) {
      existingError.remove();
    }
  }

  function setFiltersEnabled(enabled) {
    filtersEnabled = !!enabled;

    if (applyFiltersBtn) applyFiltersBtn.disabled = !filtersEnabled;
    if (clearFiltersBtn) clearFiltersBtn.disabled = !filtersEnabled;

    if (!companyListEl) return;

    companyListEl
      .querySelectorAll('input[type="checkbox"]')
      .forEach(cb => cb.disabled = !filtersEnabled);
  }

  function refreshFilterValues() {
    if (!companyListEl) return;

    companyListEl.innerHTML = '';

    if (allCompanies.length === 0) {
      companyListEl.innerHTML = '<div class="placeholder">Нет данных</div>';

      if (companyToggleWrap) {
        companyToggleWrap.innerHTML = '';
      }

      return;
    }

    allCompanies.forEach((company, idx) => {
      const id = `company_${idx}_${Math.random().toString(36).slice(2, 6)}`;

      const wrapper = document.createElement('label');
      wrapper.style.display = 'flex';
      wrapper.style.alignItems = 'center';
      wrapper.style.gap = '8px';
      wrapper.style.cursor = 'pointer';

      const cb = document.createElement('input');
      cb.type = 'checkbox';
      cb.value = company;
      cb.id = id;
      cb.disabled = !filtersEnabled;

      const span = document.createElement('span');
      span.textContent = company;

      wrapper.appendChild(cb);
      wrapper.appendChild(span);
      companyListEl.appendChild(wrapper);
    });

    if (!companyToggleWrap) return;

    if (allCompanies.length > 6) {
      companyListEl.style.maxHeight = '160px';
      companyToggleWrap.innerHTML = '';

      const toggleBtn = document.createElement('button');
      toggleBtn.className = 'btn';
      toggleBtn.type = 'button';
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

  function applyFilters(userTriggered = false) {
    if (allItems.length === 0) {
      if (userTriggered) {
        renderCards([], false);
      }

      return [];
    }

    let filtered = [...allItems];

    const priceValue = priceInput ? Number(priceInput.value) : 0;

    if (priceValue > 0) {
      filtered = filtered.filter(it => Number(it.price || 0) <= priceValue);
    }

    const capacityValue = capacityInput ? Number(capacityInput.value) : 0;

    if (capacityValue > 0) {
      filtered = filtered.filter(it => Number(it.capacity || 0) >= capacityValue);
    }

    const selectedCompanies = companyListEl
      ? Array.from(companyListEl.querySelectorAll('input[type="checkbox"]:checked'))
        .map(cb => cb.value.trim())
        .filter(Boolean)
      : [];

    if (selectedCompanies.length > 0) {
      filtered = filtered.filter(it => selectedCompanies.includes(it.company));
    }

    const filtersActive =
      priceValue > 0 ||
      capacityValue > 0 ||
      selectedCompanies.length > 0;

    filtering = filtersActive;

    if (userTriggered) {
      renderCards(filtered, !filtering && hasMore);
    }

    return filtered;
  }

  function renderCards(items, attachSentinel = true) {
    if (!cardsWrapper) return;

    cardsWrapper.innerHTML = '';

    if (!items || items.length === 0) {
      const empty = document.createElement('div');
      empty.className = 'empty';
      empty.textContent = 'Ничего не найдено по выбранным фильтрам';
      cardsWrapper.appendChild(empty);
      ensureSentinel(false);
      return;
    }

    for (const it of items) {
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
      img.src = it.image || '/shared/placeholders/room-placeholder.svg';

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
    }

    ensureSentinel(attachSentinel);
  }

  function ensureSentinel(enable = true) {
    if (!cardsWrapper) return;

    const oldSentinel = cardsWrapper.querySelector('#sentinel');

    if (oldSentinel) {
      oldSentinel.remove();
    }

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

    if (infiniteScrollObserver) {
      infiniteScrollObserver.disconnect();
    }

    infiniteScrollObserver = new IntersectionObserver(entries => {
      for (const entry of entries) {
        if (entry.isIntersecting && !isLoading && hasMore && !filtering) {
          loadMoreRooms(false);
        }
      }
    }, {
      root: null,
      rootMargin: '0px',
      threshold: 0.1
    });

    infiniteScrollObserver.observe(sentinel);
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
      retryBtn.type = 'button';
      retryBtn.textContent = 'Попробовать сейчас';

      retryBtn.addEventListener('click', () => {
        loadInitialData();
      });

      btnGroup.appendChild(retryBtn);
      container.appendChild(btnGroup);
    }

    cardsWrapper.appendChild(container);
  }

  function showStatusBar(text) {
    if (!statusBar) return;

    statusBar.hidden = false;
    statusBar.textContent = text;
  }

  function hideStatusBar() {
    if (!statusBar) return;

    statusBar.hidden = true;
    statusBar.textContent = '';
  }

  function navigate(url) {
    window.location.href = url;
  }
})();