<template>
  <div class="shell">
    <header class="header">
      <strong>Drawing Board</strong>
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
  border-bottom: 1px solid #ddd;
}
</style>
