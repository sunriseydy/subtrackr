// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * i18n coverage test: for each supported language, load every primary user page
 * and assert specific translated strings appear (positive check) AND specific
 * sentinel English-only strings do NOT leak through (negative check).
 *
 * This catches regressions where:
 *   - A new template string is added without a translation key
 *   - A translation key is wired up in English but missing in another language
 *   - Someone breaks the JSON catalog so the server falls back to keys
 *
 * Add new entries to `pages` as more surfaces get translated.
 */

const pages = [
  {
    path: '/dashboard',
    expect: {
      es: ['Panel', 'Suscripciones', 'Análisis', 'Gasto mensual', 'Gasto anual', 'Suscripciones activas', 'Ahorro mensual', 'Alle abonnementen|Todas las suscripciones|Alle Abonnements|All Subscriptions'],
      de: ['Übersicht', 'Abonnements', 'Analyse', 'Monatliche Ausgaben', 'Jährliche Ausgaben', 'Aktive Abonnements', 'Monatliche Ersparnis'],
      nl: ['Dashboard', 'Abonnementen', 'Analyse', 'Maandelijkse uitgaven', 'Jaarlijkse uitgaven', 'Actieve abonnementen'],
    },
    forbid: {
      es: ['Monthly Spend', 'Annual Spend', 'Active Subscriptions', 'Monthly Savings', 'Spending by Category', 'All Subscriptions'],
      de: ['Monthly Spend', 'Annual Spend', 'Active Subscriptions', 'Monthly Savings', 'Spending by Category', 'All Subscriptions'],
      nl: ['Monthly Spend', 'Annual Spend', 'Active Subscriptions', 'Monthly Savings', 'Spending by Category', 'All Subscriptions'],
    },
  },
  {
    path: '/subscriptions',
    expect: {
      es: ['Suscripciones', 'Añadir suscripción', 'Buscar suscripciones...', 'Autopago', 'Nombre', 'Categoría', 'Coste'],
      de: ['Abonnements', 'Abonnement hinzufügen', 'Abonnements durchsuchen...', 'Automatisch', 'Name', 'Kategorie', 'Kosten'],
      nl: ['Abonnementen', 'Abonnement toevoegen', 'Abonnementen zoeken...', 'Automatisch', 'Naam', 'Categorie', 'Kosten'],
    },
    forbid: {
      es: ['Add Subscription', 'Renewal Date'],
      de: ['Add Subscription', 'Renewal Date'],
      nl: ['Add Subscription', 'Renewal Date'],
    },
  },
  {
    path: '/form/subscription',
    expect: {
      es: ['Añadir suscripción', 'Nombre', 'Etiqueta', 'Etiquetas', 'Compartido con', 'Frecuencia', 'Coste', 'Pago automático', 'Notas'],
      de: ['Abonnement hinzufügen', 'Name', 'Etikett', 'Tags', 'Geteilt mit', 'Frequenz', 'Kosten', 'Automatische Zahlung', 'Notizen'],
      nl: ['Abonnement toevoegen', 'Naam', 'Label', 'Tags', 'Gedeeld met', 'Frequentie', 'Kosten', 'Automatische betaling', 'Notities'],
    },
    forbid: {
      es: ['Add Subscription', 'Shared with', 'Schedule *', 'Notes</label>'],
      de: ['Add Subscription', 'Shared with', 'Schedule *', 'Notes</label>'],
      nl: ['Add Subscription', 'Shared with', 'Schedule *', 'Notes</label>'],
    },
  },
  {
    path: '/settings',
    expect: {
      es: ['Ajustes', 'Apariencia', 'Idioma', 'Notificaciones por correo', 'Configuración SMTP', 'Notificaciones Pushover', 'Recordatorios de renovación', 'Avísame cuando', 'Aquí se enviarán', 'Probar conexión', 'Seguridad', 'Categorías', 'Claves API', 'Acerca de SubTrackr'],
      de: ['Einstellungen', 'Erscheinungsbild', 'Sprache', 'E-Mail-Benachrichtigungen', 'SMTP-Konfiguration', 'Pushover-Benachrichtigungen', 'Verlängerungserinnerungen', 'Verbindung testen', 'Sicherheit', 'Kategorien', 'API-Schlüssel', 'Über SubTrackr'],
      nl: ['Instellingen', 'Weergave', 'Taal', 'E-mailmeldingen', 'SMTP-configuratie', 'Pushover-meldingen', 'Verlengingsherinneringen', 'Verbinding testen', 'Beveiliging', 'Categorieën', 'API-sleutels', 'Over SubTrackr'],
    },
    forbid: {
      es: ['Email Notifications', 'SMTP Configuration', 'Pushover Notifications', 'High Cost Alerts', 'High Cost Threshold', 'Test Connection', 'Save SMTP Settings', 'Get notified before subscriptions', 'Alert when adding'],
      de: ['Email Notifications', 'SMTP Configuration', 'Pushover Notifications', 'High Cost Alerts', 'High Cost Threshold', 'Test Connection', 'Save SMTP Settings', 'Get notified before subscriptions', 'Alert when adding'],
      nl: ['Email Notifications', 'SMTP Configuration', 'Pushover Notifications', 'High Cost Alerts', 'High Cost Threshold', 'Test Connection', 'Save SMTP Settings', 'Get notified before subscriptions', 'Alert when adding'],
    },
  },
  {
    path: '/analytics',
    expect: {
      es: ['Análisis', 'Gasto mensual total', 'Gasto anual total', 'Análisis de coste'],
      de: ['Analyse', 'Monatliche Gesamtausgaben', 'Jährliche Gesamtausgaben', 'Kostenanalyse'],
      nl: ['Analyse', 'Totale maanduitgaven', 'Totale jaaruitgaven', 'Kostenanalyse'],
    },
    forbid: {
      es: ['Total Monthly Spend', 'Total Annual Spend', 'Cost Analysis'],
      de: ['Total Monthly Spend', 'Total Annual Spend', 'Cost Analysis'],
      nl: ['Total Monthly Spend', 'Total Annual Spend', 'Cost Analysis'],
    },
  },
  {
    path: '/calendar',
    expect: {
      es: ['Hoy'],
      de: ['Heute'],
      nl: ['Vandaag'],
    },
    forbid: {
      es: [],
      de: [],
      nl: [],
    },
  },
];

const languages = ['es', 'de', 'nl'];
let seededSubscriptionID = null;
let seededCategoryID = null;

// Language is a global application setting, so these checks must not race each other.
test.describe.configure({ mode: 'serial' });

test.beforeAll(async ({ request }) => {
  await request.post('/api/settings/language', { data: { lang: 'en' } });

  let categories = await (await request.get('/api/categories')).json();
  if (categories.length === 0) {
    const categoryResponse = await request.post('/api/categories', {
      data: { name: `i18n Coverage ${Date.now()}` },
    });
    expect(categoryResponse.ok()).toBeTruthy();
    const category = await categoryResponse.json();
    seededCategoryID = category.id;
    categories = [category];
  }

  const subscriptionResponse = await request.post('/api/subscriptions', {
    form: {
      name: `i18n Coverage ${Date.now()}`,
      cost: '1.00',
      schedule: 'Monthly',
      status: 'Active',
      original_currency: 'USD',
      category_id: String(categories[0].id),
      autopay: 'true',
    },
  });
  expect(subscriptionResponse.ok()).toBeTruthy();
  seededSubscriptionID = (await subscriptionResponse.json()).id;
});

test.afterAll(async ({ request }) => {
  if (seededSubscriptionID !== null) {
    await request.delete(`/api/subscriptions/${seededSubscriptionID}`);
  }
  if (seededCategoryID !== null) {
    await request.delete(`/api/categories/${seededCategoryID}`);
  }
  await request.post('/api/settings/language', { data: { lang: 'en' } });
});

for (const lang of languages) {
  test.describe(`i18n coverage — ${lang}`, () => {
    test.beforeEach(async ({ request }) => {
      const res = await request.post('/api/settings/language', { data: { lang } });
      expect(res.ok()).toBeTruthy();
    });

    for (const page of pages) {
      test(`${page.path} renders ${lang} translations`, async ({ page: browserPage }) => {
        await browserPage.goto(page.path);
        await browserPage.waitForLoadState('networkidle');
        const html = await browserPage.content();

        // Positive checks: every expected translated string is present somewhere on the page
        for (const expected of (page.expect[lang] || [])) {
          // Support OR-separated alternatives via |
          const alternatives = expected.split('|');
          const found = alternatives.some(alt => html.includes(alt));
          expect(found, `expected one of [${alternatives.join(', ')}] on ${page.path} (${lang})`).toBeTruthy();
        }

        // Negative checks: forbidden English strings must NOT appear
        for (const forbidden of (page.forbid[lang] || [])) {
          expect(
            html.includes(forbidden),
            `forbidden English string "${forbidden}" leaked through on ${page.path} (${lang})`
          ).toBeFalsy();
        }
      });
    }
  });
}
