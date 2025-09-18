package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"database/sql"
	"encoding/json"
	"net/http"
	"path/filepath"
	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
		MsjResErrInterno = "Error interno"
		MsjResAltaExito = "Alta exitosa"
		MsjResExito = "Peticion exitosa"
		MsjResErrArchivo = "No se encontro el archivo"
)

type Config struct {
		ServerPort int `json:"server_port"`
		DBHost string `json:"db_host"`
		DBPort int `json:"db_port"`
		DBNombre string `json:"db"`
		DBUsuario string `json:"db_usuario"`
		DBContra string `json:"db_contra"`
		Destino string `json:"destino"`
}

type Archivo struct {
		Id int `json:"id" form:"id"`
		Nombre string `json:"nombre" form:"nombre"`
		NombreArchivo string `json:"nombre_archivo" form:"nombre_archivo"`
		Creado string `json:"creado" form:"creado"`
}

var config = Config{ //valores x defecto				
		ServerPort: 4567,
		DBPort: 3306,
		DBUsuario: "usuario",
		DBContra: "usuario",
		DBNombre: "pdfs",
		DBHost: "127.0.1",
		Destino: "./archivos",
}

var BD *sql.DB;

func main(){
		fmt.Print("Iniciando server...")
		file, err := os.Open("config.json")
		if err == nil{				
				defer file.Close()				
				decoder := json.NewDecoder(file)
				err = decoder.Decode(&config)
				if err != nil{
						fmt.Printf("Error: %s", err)
						return
				}
		}
		if err := os.MkdirAll(config.Destino, 0755); err != nil {
				fmt.Printf("Error creando directorio: %s  %s",config.Destino, err)
				return
		}
		
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", config.DBUsuario, config.DBContra, config.DBHost, config.DBPort, config.DBNombre)
		BD, err = sql.Open("mysql", dsn)
		if err != nil {
				fmt.Printf("Error: %s", err)
				return
		}
		defer BD.Close()
		if err := BD.Ping(); err != nil {
				fmt.Printf("Error: %s", err)
				return
		}
		
		e:= echo.New()
		e.Use(middleware.CORS())
		e.Static("/", "static")
		e.GET("/api/listar",listar)
		e.GET("/api/:id", visualizar)
		e.POST("/api/subir", subir)
		e.Start(fmt.Sprintf(":%d",config.ServerPort))
}

func listar(c echo.Context) error {
		limite := c.QueryParam("limite")
		salto := c.QueryParam("salto")
		
		if _, err := strconv.Atoi(limite); err == nil {
				limite = "10"
		}
		if _, err := strconv.Atoi(salto); err == nil {
				salto = "0"
		}

		var cant int
		err := BD.QueryRow("SELECT COUNT(*) FROM archivo").Scan(&cant)
		if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"msj":MsjResErrInterno})
		}
		
		query := `SELECT id,nombre,nombre_archivo,creado FROM archivo LIMIT ? OFFSET ?`
		
		filas, err := BD.Query(query,limite,salto)
		if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"msj":MsjResErrInterno})
		}
		
		respuesta := []Archivo{}
		var aux Archivo
		for filas.Next(){
				if err := filas.Scan(&aux.Id, &aux.Nombre, &aux.NombreArchivo, &aux.Creado); err != nil {
						return c.JSON(http.StatusInternalServerError, map[string]string{"msj":MsjResErrInterno})
				}
				respuesta = append(respuesta, aux)
		}		
		return c.JSON(http.StatusOK, map[string]any{"msj":MsjResExito, "res":respuesta, "cant":cant})
}

func subir(c echo.Context) error {
		nombre := c.FormValue("nombre")
		file, err := c.FormFile("archivo")
		if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"msj":MsjResErrInterno})
		}

		src, err := file.Open()
		if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"msj":MsjResErrInterno})
		}
		defer src.Close()

		ruta := filepath.Join(config.Destino, file.Filename)

		//Inicio transaccion
		tx, err := BD.Begin()
		if err != nil {
				return err
		}		
		defer func(){
				if err != nil {tx.Rollback()}
		}()
		
		dst, err := os.Create(ruta)
		if err != nil {
				return err
		}

		_, err = io.Copy(dst, src)
		dst.Close()
		if err != nil {
				os.Remove(ruta)
				return err
		}
		query := "INSERT INTO archivo (nombre, nombre_archivo, ruta) VALUES (?, ?, ?)"
		_, err = tx.Exec(query , nombre, file.Filename, ruta)
		if err != nil {
				os.Remove(ruta)
				return err
		}
		if err = tx.Commit(); err != nil {
				os.Remove(ruta)
				return err
		}		
		return c.JSON(http.StatusOK, map[string]string{"msj":MsjResAltaExito})
}

func visualizar(c echo.Context) error {
		id := c.Param("id")
		query := "SELECT ruta, nombre_archivo FROM archivo WHERE id = ?"
		var ruta, nombre string
		err := BD.QueryRow(query, id).Scan(&ruta, &nombre)
		if err != nil {
				if err == sql.ErrNoRows {
						return c.JSON(http.StatusNotFound, map[string]string{"msj":MsjResErrArchivo})
				}
				return err
		}
		if _, err := os.Stat(ruta); os.IsNotExist(err) {
				return c.JSON(http.StatusNotFound, map[string]string{"msj":MsjResErrArchivo})
		}
		return c.Inline(ruta, nombre)
}
