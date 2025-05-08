const express = require('express');
const bodyParser = require('body-parser');
const app = express();
const PORT = 8083;

app.use(bodyParser.json());

const products = {
    'prod-001': { id: 'prod-001', name: 'Laptop', price: 999.99, categoryId: 'cat-001', inStock: true },
    'prod-002': { id: 'prod-002', name: 'Smartphone', price: 699.99, categoryId: 'cat-001', inStock: true },
    'prod-003': { id: 'prod-003', name: 'Headphones', price: 199.99, categoryId: 'cat-002', inStock: false }
};

const authenticate = (req, res, next) => {
    const authHeader = req.headers.authorization;
    if (!authHeader || !authHeader.startsWith('Bearer ')) {
        return res.status(401).json({ error: 'Authentication required' });
    }
    next();
};

app.get('/api/products', (req, res) => {
    if (req.query.delay === 'true') {
        return setTimeout(() => {
            res.json({ data: Object.values(products) });
        }, 10000);
    }

    res.json({ data: Object.values(products) });
});

app.get('/api/products/:id', (req, res) => {
    const product = products[req.params.id];
    if (!product) {
        return res.status(404).json({ error: 'Product not found' });
    }
    res.json(product);
});

app.post('/api/products', authenticate, (req, res) => {
    const id = `prod-${Math.floor(Math.random() * 1000)}`;
    const newProduct = { id, ...req.body };
    products[id] = newProduct;
    res.status(201).json(newProduct);
});

app.put('/api/products/:id', authenticate, (req, res) => {
    const id = req.params.id;
    if (!products[id]) {
        return res.status(404).json({ error: 'Product not found' });
    }
    products[id] = { ...products[id], ...req.body };
    res.json(products[id]);
});

app.delete('/api/products/:id', authenticate, (req, res) => {
    const id = req.params.id;
    if (!products[id]) {
        return res.status(404).json({ error: 'Product not found' });
    }
    delete products[id];
    res.status(204).end();
});

app.listen(PORT, () => {
    console.log(`Products service running on port ${PORT}`);
});
