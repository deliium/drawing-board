<script setup lang="ts">
import { computed } from 'vue'
import { useLocale } from '../composables/useLocale'
import type { Locale } from '../i18n'

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

const { locale, setLocale, t } = useLocale()

const brand = computed(() => t('brand.name'))
const navPractice = computed(() => t('nav.practice'))
const navPracticeShort = computed(() => t('nav.practiceShort'))
const navBoard = computed(() => t('nav.board'))
const navBoardShort = computed(() => t('nav.boardShort'))
const skipLabel = computed(() => t('nav.skip'))
const localeLabel = computed(() => t('locale.label'))

function onLocaleChange(ev: Event) {
  const value = (ev.target as HTMLSelectElement).value as Locale
  if (value === 'en' || value === 'ja') {
    if (isDev) console.debug('[AppShell] locale toggle', value)
    setLocale(value)
  }
}
</script>

<template>
  <div class="shell">
    <a class="skip-link" href="#main-content">{{ skipLabel }}</a>
    <header class="header">
      <strong class="brand">{{ brand }}</strong>
      <div class="header-end">
        <nav class="nav" :aria-label="brand">
          <router-link to="/practice" class="quiet">
            <span class="full">{{ navPractice }}</span>
            <span class="short">{{ navPracticeShort }}</span>
          </router-link>
          <router-link to="/practice/history" class="quiet">
            <span class="full">{{ t('nav.history') }}</span>
            <span class="short">{{ t('nav.historyShort') }}</span>
          </router-link>
          <router-link to="/" class="quiet">
            <span class="full">{{ navBoard }}</span>
            <span class="short">{{ navBoardShort }}</span>
          </router-link>
        </nav>
        <label class="locale">
          <span class="locale-label">{{ localeLabel }}</span>
          <select
            class="locale-select"
            :value="locale"
            :aria-label="localeLabel"
            @change="onLocaleChange"
          >
            <option value="en">{{ t('locale.en') }}</option>
            <option value="ja">{{ t('locale.ja') }}</option>
          </select>
        </label>
      </div>
    </header>
    <main id="main-content" class="main" tabindex="-1">
      <slot />
    </main>
  </div>
</template>

<style scoped>
.shell {
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
}

.skip-link {
  position: absolute;
  left: var(--space-3);
  top: var(--space-2);
  z-index: 100;
  padding: var(--space-2) var(--space-3);
  background: var(--paper-raised);
  color: var(--ink);
  border: 1px solid var(--rule);
  border-radius: var(--radius-sm);
  text-decoration: none;
  transform: translateY(-200%);
}

.skip-link:focus-visible {
  transform: translateY(0);
}

.header {
  padding: max(var(--space-3), env(safe-area-inset-top))
    max(var(--space-4), env(safe-area-inset-right))
    var(--space-3)
    max(var(--space-4), env(safe-area-inset-left));
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  border-bottom: 1px solid var(--rule);
  background: color-mix(in srgb, var(--paper-raised) 88%, transparent);
  flex-wrap: wrap;
}

.brand {
  font-size: 1.15rem;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--ink);
}

.header-end {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  flex-wrap: wrap;
}

.nav {
  display: flex;
  gap: var(--space-3);
  flex-wrap: wrap;
}

.quiet {
  color: var(--ink-muted);
  text-decoration: none;
  font-size: 0.95rem;
  min-height: var(--touch-min);
  display: inline-flex;
  align-items: center;
  padding: 0 var(--space-1);
}

.quiet:hover {
  text-decoration: underline;
  color: var(--ink);
}

.quiet.router-link-active {
  color: var(--ink);
  text-decoration: underline;
}

.short {
  display: none;
}

@media (max-width: 400px) {
  .full {
    display: none;
  }
  .short {
    display: inline;
  }
}

.locale {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  min-height: var(--touch-min);
}

.locale-label {
  font-size: 0.85rem;
  color: var(--ink-muted);
}

.locale-select {
  min-height: var(--touch-min);
  min-width: 6.5rem;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--rule);
  border-radius: var(--radius-sm);
  background: var(--paper-raised);
  color: var(--ink);
  font: inherit;
}

.main {
  flex: 1;
  width: 100%;
  max-width: var(--content-max);
  margin: 0 auto;
  padding: var(--space-4) max(var(--space-4), env(safe-area-inset-right))
    max(var(--space-5), env(safe-area-inset-bottom))
    max(var(--space-4), env(safe-area-inset-left));
}

/* Free board uses full width for the canvas stage */
.main:has(.board-page) {
  max-width: none;
  padding-left: max(var(--space-3), env(safe-area-inset-left));
  padding-right: max(var(--space-3), env(safe-area-inset-right));
}
</style>
