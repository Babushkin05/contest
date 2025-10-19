# 🍔 Контест по поеданию бургеров

**Живой лидерборд:** [http://contest.babushkin05.ru/](http://contest.babushkin05.ru/)

---

## 🔥 API

Все запросы защищены паролем.  
Пароль хранится в `.env` в переменной `PASS`.

Пример содержимого `.env`:

```
PASS=mysecretpass
```

---

### ⚙️ Формат запросов

```bash
# Увеличить счётчик участника на 1
curl -X POST http://contest.babushkin05.ru/{pass}/{id}/up

# Установить конкретное значение
curl -X POST http://contest.babushkin05.ru/{pass}/{id}/{count}
```

Примеры:

```bash
curl -X POST http://contest.babushkin05.ru/mysecretpass/vova/up
curl -X POST http://contest.babushkin05.ru/mysecretpass/egor/7
```

Если пароль неверный → возвращается **404 Not Found**

---

### 📊 Получить список лидеров

```bash
curl http://contest.babushkin05.ru/leaders
```

Ответ в JSON:

```json
[
  {"id": "vova", "name": "Вова", "count": 5, "photo": "/photos/vova.jpg"},
  {"id": "misha", "name": "Миша", "count": 3, "photo": "/photos/misha.jpg"}
]
```

---

## 🧍‍♂️ Список участников

| ID | Имя | Фото |
|----|-----|------|
| `vova` | Вова | /photos/vova.jpg |
| `misha` | Миша | /photos/misha.jpg |
| `stepa` | Стёпа | /photos/stepa.jpg |
| `egor` | Егор | /photos/egor.jpg |
| `timur` | Тимур | /photos/timur.jpg |
| `kp` | КатяПолина | /photos/kp.jpg |
| `timoha` | Тимоха | /photos/timoha.jpg |
| `igor` | Игорь | /photos/igor.jpg |
| `makar` | Макар | /photos/makar.jpg |

---

## 🧠 Примечания

- Фронтенд автоматически обновляет таблицу каждые 2 секунды  
- Для обновления счёта используйте **правильный пароль**  
- Если API вернул `404`, проверьте пароль или ID участника  
- Сервер работает на порту `1337` и проксируется через Nginx
