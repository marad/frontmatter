
# Cel narzędzia
Frontmatter jest narzędziem CLI do modyfikacji YAML frontmatter w plikach tekstowych z poziomu terminala.

# Obsługiwane przypadki
- potrafi dodać frontmatter jeśli go nie ma
- potrafi zmodyfikować tylko wskazane pole, bez modyfikacji pozostałych
- potrafi zwrócić zawartość YAML frontmatter
- potrafi usunąć YAML frontmatter z pliku
- flaga `--dry-run` zamiast zapisywać zmiany w pliku pokazuje jedynie zmieniony frontmatter na stdout

# Użycie narzędzia

```
frontmatter [get|set] [--dry-run] [...] <file>
```

# Przykładowe wykorzystanie

Proste ustawienie pola `message` we frontmatter na wartość `Hello World`:
```bash
    frontmatter set message="Hello World" file.md
```

Ustawienie zagnieżdżonego pola `object.field` na wartość `5`:
```bash
    frontmatter set object.field=5 file.md
```

Ustawienie dwóch pól `a` oraz `b` jednocześnie:
```bash
    frontmatter set a=1 b=value file.md
```

Wyświetlenie wartości pola `message` z frontmatter z pliku na stdout:
```bash
    frontmatter get message file.md
```

Wyświetlenie całego frontmatter z pliku na stdout:
```bash
    frontmatter get file.md
``` 

Pobranie nieistniejącego pola powinno zwrócić kod błędu 2 i nie wypisać niczego na stdout.

Pobranie frontmatter z pliku, które nie zawiera frontmatter powinno zwrócić kod błędu 2 i nie wypisywać niczego na stdout.




