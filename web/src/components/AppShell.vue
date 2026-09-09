<template>
  <div class="shell">
    <header class="header">
      <strong>Japanese Handwriting Practice</strong>
      <nav class="nav">
        <router-link to="/practice" class="quiet">Practice hiragana</router-link>
        <router-link to="/" class="quiet">Free board</router-link>
      </nav>
    </header>
    <main>
      <slot />
    </main>
  </div>
</template>

<style scoped>
.shell {
  min-height: 100vh;
}
.header {
  padding: 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border-bottom: 1px solid #ddd;
}
.nav {
  display: flex;
  gap: 14px;
}
.quiet {
  color: #475569;
  text-decoration: none;
  font-size: 0.95rem;
}
.quiet:hover {
  text-decoration: underline;
}
.quiet.router-link-active {
  color: #1e293b;
  text-decoration: underline;
}
</style>
