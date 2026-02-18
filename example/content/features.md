---
title: "Features"
---

# What your markdown can do

OpenDoc renders standard markdown, but it also supports **equations**, **syntax-highlighted code**, **embedded HTML and JavaScript**, **tabbed content**, **theorem blocks**, and **margin notes**. This page is a live demo of all of them.

---

## Equations and LaTeX

Write inline math like $E = mc^2$ or $\nabla \times \mathbf{B} = \mu_0 \mathbf{J}$ directly in your text.

For display equations, use $$ on their own lines:

$$
\int_0^\infty e^{-x^2}\, dx = \frac{\sqrt{\pi}}{2}
$$

LaTeX environments work too:

\begin{align}
\nabla \cdot \mathbf{E} &= \frac{\rho}{\epsilon_0} \\
\nabla \cdot \mathbf{B} &= 0 \\
\nabla \times \mathbf{E} &= -\frac{\partial \mathbf{B}}{\partial t} \\
\nabla \times \mathbf{B} &= \mu_0 \mathbf{J} + \mu_0 \epsilon_0 \frac{\partial \mathbf{E}}{\partial t}
\end{align}

You can even number equations for reference:

$$
F = ma
$${#eq:newton}

---

## Theorem blocks

:::theorem Pythagorean Theorem
For a right triangle with legs $a$, $b$ and hypotenuse $c$:

$$a^2 + b^2 = c^2$$
:::

:::definition Continuous Function
A function $f: \mathbb{R} \to \mathbb{R}$ is **continuous** at a point $a$ if:

$$\lim_{x \to a} f(x) = f(a)$$
:::

:::proof
Let $\epsilon > 0$. Choose $\delta = \epsilon / M$ where $M$ is the Lipschitz constant. Then for $|x - a| < \delta$ we have $|f(x) - f(a)| \leq M|x - a| < \epsilon$. $\square$
:::

Supported types: **theorem**, **definition**, **lemma**, **proposition**, **corollary**, **remark**, **proof**. Each type is auto-numbered independently.

---

## Code with syntax highlighting

```python
def fibonacci(n: int) -> list[int]:
    """Generate the first n Fibonacci numbers."""
    seq = [0, 1]
    for _ in range(n - 2):
        seq.append(seq[-1] + seq[-2])
    return seq[:n]

print(fibonacci(10))  # [0, 1, 1, 2, 3, 5, 8, 13, 21, 34]
```

```javascript
const debounce = (fn, ms) => {
  let timer;
  return (...args) => {
    clearTimeout(timer);
    timer = setTimeout(() => fn(...args), ms);
  };
};
```

```sql
SELECT users.name, COUNT(posts.id) AS post_count
FROM users
LEFT JOIN posts ON posts.author_id = users.id
GROUP BY users.id
ORDER BY post_count DESC
LIMIT 10;
```

All common languages are supported: Python, JavaScript, TypeScript, Go, Rust, SQL, HTML, CSS, YAML, and many more.

---

## Tabs

:::tabs
=== Python
```python
for i in range(5):
    print(f"Hello {i}")
```
=== JavaScript
```javascript
for (let i = 0; i < 5; i++) {
  console.log(`Hello ${i}`);
}
```
=== Go
```go
for i := 0; i < 5; i++ {
    fmt.Printf("Hello %d\n", i)
}
```
:::

Content inside each tab is full markdown — code blocks, lists, images, everything works.

---

## Margin notes

:::sidenote Why margin notes?
Margin notes keep supplementary information visible without interrupting the main flow. They are inspired by Edward Tufte's book designs.
:::

Margin notes appear alongside your text. They are perfect for definitions, historical context, or tangential thoughts that would clutter the main body.

:::deepdive How OpenDoc renders this
The `:::sidenote` syntax is preprocessed before the markdown is converted to HTML. The inner content is rendered as full markdown, so you can include **bold**, *italic*, code, and even math like $e^{i\pi} + 1 = 0$ inside a note.
:::

Variants: **sidenote** (general), **deepdive** (extended detail), **aside** (tangential), **widget** (interactive).

---

## Embedded HTML and JavaScript

Raw HTML passes straight through. Build anything you can build on the web:

<div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 2rem; border-radius: 12px; color: white; text-align: center; margin: 1.5rem 0;">
  <h3 style="margin: 0 0 0.5rem; color: white;">Custom styled section</h3>
  <p style="margin: 0; opacity: 0.9;">This is raw HTML with inline CSS — gradients, rounded corners, anything goes.</p>
</div>

You can embed interactive JavaScript too:

<div id="counter-demo" style="display: flex; align-items: center; gap: 1rem; padding: 1rem; border: 1px solid #333; border-radius: 8px; margin: 1.5rem 0;">
  <button onclick="document.getElementById('count').textContent = parseInt(document.getElementById('count').textContent) - 1" style="padding: 0.5rem 1rem; border-radius: 6px; border: 1px solid #555; background: #2a2a3a; color: #fff; cursor: pointer; font-size: 1.1rem;">−</button>
  <span id="count" style="font-size: 1.5rem; font-weight: bold; min-width: 3rem; text-align: center;">0</span>
  <button onclick="document.getElementById('count').textContent = parseInt(document.getElementById('count').textContent) + 1" style="padding: 0.5rem 1rem; border-radius: 6px; border: 1px solid #555; background: #2a2a3a; color: #fff; cursor: pointer; font-size: 1.1rem;">+</button>
  <span style="opacity: 0.6; font-size: 0.85rem;">← try clicking these</span>
</div>

This means you can embed **charts**, **calculators**, **interactive diagrams**, **videos**, **iframes** — anything that works in HTML.

---

## Tables, task lists, and more

| Feature         | Syntax                  | Supported |
|-----------------|-------------------------|-----------|
| Bold            | `**text**`              | Yes       |
| Italic          | `*text*`                | Yes       |
| Strikethrough   | `~~text~~`              | Yes       |
| Task lists      | `- [x] done`            | Yes       |
| Auto-link URLs  | just paste the URL      | Yes       |
| Heading anchors | auto-generated          | Yes       |
| Smart quotes    | `"text"` → "text"       | Yes       |

- [x] Markdown basics
- [x] Equations and LaTeX
- [x] Syntax-highlighted code
- [x] Embedded HTML and JavaScript
- [x] Tabs, theorems, and margin notes
- [ ] Your content here!

---

## Edit this page

This file lives at `content/features.md`. Open it in your editor, or ask the chatbot:

> "Edit the features page and add a section about my project"

Every page on your site works the same way — it is just a markdown file.
