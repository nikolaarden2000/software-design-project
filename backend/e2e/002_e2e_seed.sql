-- Данные только для E2E-тестов.
-- Пароль для всех тестовых пользователей: Password123

INSERT INTO companies (id, name, description) VALUES
(1, 'E2E Компания', 'Компания для запуска E2E-тестов');

INSERT INTO locations (
  id,
  company_id,
  city,
  street,
  house_number,
  latitude,
  longitude,
  timezone
) VALUES
(
  1,
  1,
  'Москва',
  'E2E улица',
  '1',
  55.751244,
  37.618423,
  'Europe/Moscow'
);

INSERT INTO users (id, email, name, password_hash, role) VALUES
(
  1,
  'test3@test3.ru',
  'E2E Пользователь',
  '1$1$65536$4$32$00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff$b264db18ed4ce3340c4f3f3b31f619d926cb6f87f0a29f5b00e972ba6579e95b',
  'user'
),
(
  2,
  'superuser@mail.com',
  'E2E Суперпользователь',
  '1$1$65536$4$32$00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff$b264db18ed4ce3340c4f3f3b31f619d926cb6f87f0a29f5b00e972ba6579e95b',
  'superuser'
),
(
  3,
  'admin-abc@mail.com',
  'E2E Администратор',
  '1$1$65536$4$32$00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff$b264db18ed4ce3340c4f3f3b31f619d926cb6f87f0a29f5b00e972ba6579e95b',
  'admin'
);

INSERT INTO admin_locations (admin_id, location_id) VALUES
(3, 1);

INSERT INTO rooms (
  id,
  location_id,
  title,
  capacity,
  price,
  description,
  images,
  available_from,
  available_to,
  status,
  created_by
) VALUES
(
  1,
  1,
  'E2E каталог переговорная',
  8,
  1500,
  'Опубликованная комната для проверки каталога и карточки помещения',
  ARRAY['/shared/placeholders/room-placeholder.svg'],
  '09:00',
  '21:00',
  'published',
  3
),
(
  2,
  1,
  'E2E комната для бронирования',
  6,
  1200,
  'Опубликованная комната без броней для сценария бронирования пользователем',
  ARRAY['/shared/placeholders/room-placeholder.svg'],
  '09:00',
  '21:00',
  'published',
  3
),
(
  3,
  1,
  'E2E комната для отмены брони',
  5,
  1300,
  'Опубликованная комната без броней для сценария отмены бронирования',
  ARRAY['/shared/placeholders/room-placeholder.svg'],
  '09:00',
  '21:00',
  'published',
  3
),
(
  4,
  1,
  'E2E черновик для отправки на модерацию',
  10,
  1800,
  'Черновик помещения для проверки отправки на модерацию',
  ARRAY['/shared/placeholders/room-placeholder.svg'],
  '09:00',
  '21:00',
  'draft',
  3
),
(
  5,
  1,
  'E2E pending для одобрения',
  12,
  2200,
  'Помещение на модерации для проверки одобрения суперпользователем',
  ARRAY['/shared/placeholders/room-placeholder.svg'],
  '09:00',
  '21:00',
  'pending',
  3
),
(
  6,
  1,
  'E2E pending для отклонения',
  7,
  1600,
  'Помещение на модерации для проверки отклонения суперпользователем',
  ARRAY['/shared/placeholders/room-placeholder.svg'],
  '09:00',
  '21:00',
  'pending',
  3
),
(
  7,
  1,
  'E2E комната с бронью для администратора',
  8,
  1700,
  'Опубликованная комната с будущей бронью для проверки отмены бронирования администратором',
  ARRAY['/shared/placeholders/room-placeholder.svg'],
  '09:00',
  '21:00',
  'published',
  3
),
(
  8,
  1,
  'E2E комната с будущей бронью',
  8,
  1900,
  'Опубликованная комната с будущей бронью для проверки запрета немедленного архивирования',
  ARRAY['/shared/placeholders/room-placeholder.svg'],
  '09:00',
  '21:00',
  'published',
  3
),
(
  9,
  1,
  'E2E комната без будущих броней',
  4,
  1000,
  'Опубликованная комната без будущих броней для проверки немедленного архивирования',
  ARRAY['/shared/placeholders/room-placeholder.svg'],
  '09:00',
  '21:00',
  'published',
  3
);

INSERT INTO bookings (
  id,
  user_id,
  room_id,
  start_time,
  end_time,
  status,
  total_price
) VALUES
(
  1,
  1,
  7,
  date_trunc('day', now() + interval '2 days') + interval '10 hours',
  date_trunc('day', now() + interval '2 days') + interval '11 hours',
  'booked',
  1700
),
(
  2,
  1,
  8,
  date_trunc('day', now() + interval '3 days') + interval '12 hours',
  date_trunc('day', now() + interval '3 days') + interval '13 hours',
  'booked',
  1900
);

SELECT setval('companies_id_seq', 1, true);
SELECT setval('locations_id_seq', 1, true);
SELECT setval('users_id_seq', 3, true);
SELECT setval('rooms_id_seq', 9, true);
SELECT setval('bookings_id_seq', 2, true);