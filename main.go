package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"

	"github.com/numanblue/logingolang/controllers"

	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"
)

type User struct {
	ID       int
	Username string
	Password string // Perhatikan: password harus dihash sebelum disimpan (tidak dilakukan di contoh ini)
}

var db *sql.DB

func main() {
	// Setup database connection
	var err error
	db, err = sql.Open("mysql", "root:@tcp(localhost:3306)/myappdb")
	if err != nil {
		fmt.Println("Failed to connect to database:", err)
		return
	}
	defer db.Close()

	// Create a new Echo instance
	e := echo.New()

	// Set up template
	t := &Template{
		templates: template.Must(template.ParseGlob("templates/*.html")),
	}
	e.Renderer = t

	// Routes
	e.GET("/", beranda)

	e.GET("/login", controllers.ShowLoginPage)
	e.POST("/login", handleLogin)
	e.GET("/protected", protectedHandler)
	e.GET("/add-product", showAddProductPage)
	e.POST("/products", createProduct)
	e.PUT("/products/:id", updateProduct)
	e.DELETE("/products/:id", deleteProduct)
	e.GET("/products", showProductsPage)
	e.GET("/products/:id/edit", showEditProductPage)

	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func beranda(c echo.Context) error {
	return c.Render(http.StatusOK, "login.html", nil)
}

func handleLogin(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	// Query ke database untuk mencari pengguna dengan username yang sesuai
	var user User
	err := db.QueryRow("SELECT id, username, password FROM users WHERE username=?", username).Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		// Jika username tidak ditemukan, atau ada kesalahan lain dalam query, tampilkan pesan kesalahan di halaman login
		return c.Render(http.StatusOK, "login.html", map[string]interface{}{
			"Message": "Invalid username or password.",
		})
	}

	// Periksa apakah password yang dimasukkan cocok dengan password yang ada di database
	if user.Password != password {
		// Jika password tidak cocok, tampilkan pesan kesalahan di halaman login
		return c.Render(http.StatusOK, "login.html", map[string]interface{}{
			"Message": "Invalid username or password.",
		})
	}

	// Jika login berhasil, atur cookie sesi dan arahkan pengguna ke halaman terproteksi
	cookie := &http.Cookie{
		Name:     "sessionID",
		Value:    strconv.Itoa(user.ID),
		Path:     "/",
		HttpOnly: true,
	}
	c.SetCookie(cookie)
	return c.Redirect(http.StatusSeeOther, "/protected")
}

func protectedHandler(c echo.Context) error {
	sessionID, err := c.Cookie("sessionID")
	if err != nil || sessionID.Value == "" {
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	// Query ke database untuk mendapatkan informasi pengguna berdasarkan ID sesi yang ada di cookie
	var user User
	err = db.QueryRow("SELECT id, username FROM users WHERE id=?", sessionID.Value).Scan(&user.ID, &user.Username)
	if err != nil {
		// Jika ada kesalahan dalam query, atau ID sesi tidak ditemukan, arahkan kembali ke halaman login
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	// Jika pengguna sudah login, tampilkan halaman terproteksi
	return c.Render(http.StatusOK, "protected.html", map[string]interface{}{
		"Username": user.Username,
	})
}

type Product struct {
	ID          int
	Name        string
	Description string
	Price       float64
}

func createProduct(c echo.Context) error {
	// Ambil data produk dari form menggunakan c.FormValue()
	name := c.FormValue("name")
	description := c.FormValue("description")
	priceStr := c.FormValue("price")

	// Konversi harga dari string ke float64
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid price")
	}

	// Buat variabel untuk menampung data produk dari form
	p := Product{
		Name:        name,
		Description: description,
		Price:       price,
	}

	// Query ke database untuk menyimpan produk baru
	_, err = db.Exec("INSERT INTO products (name, description, price) VALUES (?, ?, ?)", p.Name, p.Description, p.Price)
	if err != nil {
		fmt.Println("Failed to insert product:", err)
		return err
	}

	// Redirect ke halaman daftar produk setelah berhasil menyimpan produk
	return c.Redirect(http.StatusSeeOther, "/products")
}

func updateProduct(c echo.Context) error {
	// Ambil ID produk dari URL parameter
	productIDStr := c.Param("id")

	// Konversi ID produk dari string ke int
	productID, err := strconv.Atoi(productIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid product ID")
	}

	// Ambil data produk dari form menggunakan c.FormValue()
	name := c.FormValue("name")
	description := c.FormValue("description")
	priceStr := c.FormValue("price")

	// Konversi harga dari string ke float64
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid price")
	}

	// Update produk di database berdasarkan ID
	_, err = db.Exec("UPDATE products SET name=?, description=?, price=? WHERE id=?", name, description, price, productID)
	if err != nil {
		fmt.Println("Failed to update product:", err)
		return err
	}

	// Redirect ke halaman daftar produk setelah berhasil mengupdate produk
	return c.Redirect(http.StatusSeeOther, "/products")
}

func deleteProduct(c echo.Context) error {
	productID := c.Param("id")

	// Hapus produk dari database berdasarkan ID
	_, err := db.Exec("DELETE FROM products WHERE id=?", productID)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func showProductsPage(c echo.Context) error {
	// Query ke database untuk mengambil semua produk
	rows, err := db.Query("SELECT id, name, description, price FROM products")
	if err != nil {
		return err
	}
	defer rows.Close()

	products := []Product{}
	for rows.Next() {
		var p Product
		err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price)
		if err != nil {
			return err
		}
		products = append(products, p)
	}

	return c.Render(http.StatusOK, "products.html", products)
}

func showEditProductPage(c echo.Context) error {
	productID := c.Param("id")

	// Query ke database untuk mengambil produk berdasarkan ID
	var p Product
	err := db.QueryRow("SELECT id, name, description, price FROM products WHERE id=?", productID).Scan(&p.ID, &p.Name, &p.Description, &p.Price)
	if err != nil {
		return err
	}

	return c.Render(http.StatusOK, "edit_product.html", p)
}
func showAddProductPage(c echo.Context) error {
	return c.Render(http.StatusOK, "add_product.html", nil)
}
