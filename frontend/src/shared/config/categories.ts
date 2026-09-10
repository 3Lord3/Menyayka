// The backend has no category-dictionary endpoint: the category is stored as text, so the list
// lives here. Free input isn't allowed — clustering compares categories by exact string equality,
// and "Electronics" and "electronics" would end up in different candidate pools.
export const ITEM_CATEGORIES = [
  'Одежда, обувь, аксессуары',
  'Часы и украшения',
  'Сумки и рюкзаки',
  'Детские товары',
  'Игрушки и хобби',
  'Настольные игры',
  'Книги и журналы',
  'Музыкальные инструменты',
  'Товары для животных',
  'Спорт и отдых',
  'Велосипеды',
  'Туризм и активный отдых',
  'Мебель и интерьер',
  'Бытовая техника',
  'Посуда и кухонные принадлежности',
  'Текстиль и постельное бельё',
  'Ремонт и строительство',
  'Инструменты',
  'Электрика и освещение',
  'Сантехника',
  'Стройматериалы',
  'Садовая техника',
  'Растения и сад',
  'Запчасти',
  'Экипировка',
  'Шины, диски и колёса',
  'Аксессуары',
  'Телефоны',
  'Ноутбуки',
  'Планшеты и электронные книги',
  'Телевизоры и проекторы',
  'Компьютеры и комплектующие',
  'Оргтехника и расходники',
  'Фототехника',
  'Игры и приставки',
  'Аудио- и видеотехника',
  'Аудио- и видеотехника для авто',
] as const;

// the DB column is VARCHAR(100); the backend doesn't validate overflow and returns 500
export const CATEGORY_MAX_LENGTH = 100;

// SCRUM-52 requirement; records created before it come with a shorter description, so the
// item and request forms validate length explicitly — antd doesn't validate initialValues
export const DESCRIPTION_MIN_LENGTH = 25;

// a server category may be missing from the dictionary (data older than the migration):
// without its own option, editing would silently reset the value and with it the matching pool
export function categoryOptions(current?: string): { value: string; label: string }[] {
  const options = ITEM_CATEGORIES.map((name) => ({ value: name, label: name }));
  if (current && !ITEM_CATEGORIES.some((name) => name === current)) {
    return [{ value: current, label: current }, ...options];
  }
  return options;
}
